# Milestone 09A: Template and profile contracts

## Scope

This documentation-only milestone fixes the experimental `v1alpha1` contracts
for the packaged catalog, skill templates, project profiles, and deterministic
composition plans. It also reserves consistent CLI families and help behavior.

## Decisions carried forward

- The first-party catalog is immutable, versioned with the CLI, and offline.
- Templates use fixed output paths and single-pass substitution of declared
  typed variables in UTF-8 text resources only.
- Profiles select exact template versions and bind variables without
  expressions.
- Profile recommendation is read-only evidence; selection remains explicit.
- New-project creation and existing-workspace composition share one plan schema
  but have distinct preconditions.
- Plans expose every output, conflict, operation, owner, and digest before any
  mutation.
- Provider output, Spec Kit initialization, execution, first-party catalog
  content, and remote catalogs remain outside this milestone.

## Next approval gates

1. Implement strict catalog, template, and profile parsers plus read-only
   inspection.
2. Implement bounded authoring scaffold creation in an explicit absent path.
3. Implement deterministic rendering and non-executable composition planning.
4. Approve transactional composition execution and recovery separately.
5. Review and approve each first-party template and pilot profile as product
   content.
