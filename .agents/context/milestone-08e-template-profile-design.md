# Milestone 08E: Template and profile composition design

## Approved boundary

This documentation-only slice defines the relationship between skill templates,
project profiles, rendered canonical skills, and downstream provider output. It
adds no executable command, schema implementation, catalog content, dependency,
or workspace mutation.

## Material facts

- The current CLI manages complete canonical skill packages but does not author
  them or select a project-specific set.
- Catalog templates must not live under installed `.agents/skills/` paths.
- The initial catalog is versioned with and distributed inside the CLI so that
  rendering is offline and reproducible.
- Project inspection may recommend profiles, but the user must select a profile
  explicitly before a plan can be approved.
- Template substitution is declarative and bounded; arbitrary evaluation,
  environment expansion, scripts, network access, and credential access remain
  outside the design.
- New-project creation and existing-workspace composition are separate use
  cases with a shared deterministic planning and approval boundary.
- Provider synchronization moves after canonical template and profile
  composition in the planned release sequence.

## Remaining gates

1. Specify the exact template, profile, catalog, and composition-plan schemas.
2. Approve an authoring scaffold with an explicit absent output destination and
   the rendering implementation.
3. Approve project inspection and deterministic composition planning.
4. Approve composition execution, recovery, and ownership evidence.
5. Approve the initial first-party templates and a pilot profile independently.
