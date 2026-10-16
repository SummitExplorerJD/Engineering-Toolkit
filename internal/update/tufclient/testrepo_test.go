package tufclient

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	engupdate "github.com/SummitExplorerJD/Engineering-Toolkit/internal/update"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

type testRepository struct {
	server     *httptest.Server
	root       []byte
	targetPath string
	target     []byte
	mu         sync.RWMutex
	responses  map[string][]byte
	keys       map[string]ed25519.PrivateKey
	timestamp1 []byte
	targetURL  string
}

func newTestRepository(t *testing.T, expires time.Time) *testRepository {
	t.Helper()
	keys := make(map[string]ed25519.PrivateKey)
	root := metadata.Root(expires)
	root.Signed.ConsistentSnapshot = true
	for _, role := range []string{metadata.ROOT, metadata.TIMESTAMP, metadata.SNAPSHOT, metadata.TARGETS} {
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		keys[role] = privateKey
		publicKey, err := metadata.KeyFromPublicKey(privateKey.Public())
		if err != nil {
			t.Fatal(err)
		}
		if err := root.Signed.AddKey(publicKey, role); err != nil {
			t.Fatal(err)
		}
	}
	signRole(t, root, keys[metadata.ROOT])
	rootBytes := roleBytes(t, root)

	targetPath := "releases/stable/v1.2.3/windows-amd64/engkit.exe"
	targetBytes := []byte("verified engineering toolkit executable")
	targetInfo, err := metadata.TargetFile().FromBytes(targetPath, targetBytes, "sha256")
	if err != nil {
		t.Fatal(err)
	}
	custom, err := json.Marshal(map[string]any{
		"schema": engupdate.TargetSchema, "version": "v1.2.3", "channel": engupdate.StableChannel,
		"os": "windows", "arch": "amd64", "format": engupdate.ExecutableFormatRaw,
	})
	if err != nil {
		t.Fatal(err)
	}
	rawCustom := json.RawMessage(custom)
	targetInfo.Custom = &rawCustom
	targets := metadata.Targets(expires)
	targets.Signed.Targets[targetPath] = targetInfo
	signRole(t, targets, keys[metadata.TARGETS])
	targetsBytes := roleBytes(t, targets)

	snapshot := metadata.Snapshot(expires)
	snapshot.Signed.Meta["targets.json"] = metaFile(targets.Signed.Version, targetsBytes)
	signRole(t, snapshot, keys[metadata.SNAPSHOT])
	snapshotBytes := roleBytes(t, snapshot)

	timestamp := metadata.Timestamp(expires)
	timestamp.Signed.Meta["snapshot.json"] = metaFile(snapshot.Signed.Version, snapshotBytes)
	signRole(t, timestamp, keys[metadata.TIMESTAMP])
	timestampBytes := roleBytes(t, timestamp)

	digest := sha256.Sum256(targetBytes)
	targetLeaf := fmt.Sprintf("/targets/releases/stable/v1.2.3/windows-amd64/%s.engkit.exe", hex.EncodeToString(digest[:]))
	repository := &testRepository{
		root: rootBytes, targetPath: targetPath, target: targetBytes,
		keys: keys, timestamp1: timestampBytes, targetURL: targetLeaf,
		responses: map[string][]byte{
			"/metadata/timestamp.json":  timestampBytes,
			"/metadata/1.snapshot.json": snapshotBytes,
			"/metadata/1.targets.json":  targetsBytes,
			targetLeaf:                  targetBytes,
		},
	}
	repository.server = httptest.NewTLSServer(http.HandlerFunc(repository.serveHTTP))
	t.Cleanup(repository.server.Close)
	return repository
}

func (r *testRepository) setRootRotation(t *testing.T, version int64, signatures string) {
	t.Helper()
	rotated, err := metadata.Root().FromBytes(r.root)
	if err != nil {
		t.Fatal(err)
	}
	rotated.Signed.Version = version
	rotated.ClearSignatures()
	_, nextKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := metadata.KeyFromPublicKey(nextKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	if err := rotated.Signed.AddKey(publicKey, metadata.ROOT); err != nil {
		t.Fatal(err)
	}
	rotated.Signed.Roles[metadata.ROOT].Threshold = 2
	signRole(t, rotated, r.keys[metadata.ROOT])
	switch signatures {
	case "dual":
		signRole(t, rotated, nextKey)
	case "duplicate":
		rotated.Signatures = append(rotated.Signatures, rotated.Signatures[0])
	case "insufficient":
	default:
		t.Fatalf("unknown signature mode %q", signatures)
	}
	r.mu.Lock()
	r.responses["/metadata/2.root.json"] = roleBytes(t, rotated)
	r.mu.Unlock()
}

func (r *testRepository) advanceTimestamp(t *testing.T) {
	t.Helper()
	timestamp, err := metadata.Timestamp().FromBytes(r.timestamp1)
	if err != nil {
		t.Fatal(err)
	}
	timestamp.Signed.Version = 2
	timestamp.ClearSignatures()
	signRole(t, timestamp, r.keys[metadata.TIMESTAMP])
	r.mu.Lock()
	r.responses["/metadata/timestamp.json"] = roleBytes(t, timestamp)
	r.mu.Unlock()
}

func (r *testRepository) rollbackTimestamp() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses["/metadata/timestamp.json"] = r.timestamp1
}

func (r *testRepository) replaceTarget(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responses[r.targetURL] = data
}

func (r *testRepository) serveHTTP(w http.ResponseWriter, request *http.Request) {
	r.mu.RLock()
	data, ok := r.responses[request.URL.Path]
	r.mu.RUnlock()
	if !ok {
		http.NotFound(w, request)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	_, _ = w.Write(data)
}

func (r *testRepository) corruptTimestamp(t *testing.T) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var envelope map[string]any
	if err := json.Unmarshal(r.responses["/metadata/timestamp.json"], &envelope); err != nil {
		t.Fatal(err)
	}
	signatures := envelope["signatures"].([]any)
	signature := signatures[0].(map[string]any)
	signature["sig"] = "00"
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	r.responses["/metadata/timestamp.json"] = data
}

func signRole[T metadata.Roles](t *testing.T, role *metadata.Metadata[T], privateKey ed25519.PrivateKey) {
	t.Helper()
	payload, err := json.Marshal(role.Signed)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := metadata.KeyFromPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	keyID, err := publicKey.ID()
	if err != nil {
		t.Fatal(err)
	}
	role.Signatures = append(role.Signatures, metadata.Signature{
		KeyID: keyID, Signature: metadata.HexBytes(ed25519.Sign(privateKey, payload)),
	})
}

func roleBytes[T metadata.Roles](t *testing.T, role *metadata.Metadata[T]) []byte {
	t.Helper()
	data, err := role.ToBytes(false)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func metaFile(version int64, data []byte) *metadata.MetaFiles {
	digest := sha256.Sum256(data)
	return &metadata.MetaFiles{
		Version: version, Length: int64(len(data)),
		Hashes: metadata.Hashes{"sha256": digest[:]},
	}
}
