# Material Change Approval Workflow

## 1. Discover

Read applicable instructions and inspect the live repository without mutation. Identify existing user changes and authoritative documents.

## 2. Propose

Present:

- Objective and bounded scope.
- Exact files or external resources affected.
- Requirements and decisions implemented.
- Dependencies and schema changes.
- Test and verification plan.
- Security, compatibility, recovery, and rollback considerations.
- Explicit non-goals.

## 3. Approve

Wait for unambiguous human approval. Approval applies only to the presented scope. A changed plan, new dependency, expanded path set, external mutation, Git operation, or destructive action requires renewed approval.

## 4. Execute

Preserve unrelated work. Make the smallest coherent change. Record material deviations and stop if an assumption invalidates the approved approach.

## 5. Verify

Run the approved checks, inspect changed files, and distinguish observed evidence from untested expectations.

## 6. Handoff

Report changed files, results, limitations, remaining decisions, and rollback guidance. Stop before the next milestone, commit, push, deployment, or publication.

