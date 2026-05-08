This directory preserves the original V1.0 incremental migration chain that previously lived in the root `server/migrations/` directory.

Current active bootstrap path:

- `000001_init_mysql_schema`
- `000002_seed_initial_auth_data`
- `000003_task_execution_security`
- `000004_v07_webhook_security`
- `000005_v10_productization_bundle`

Why this archive exists:

- New environment setup now uses a shorter root migration chain.
- The original `000005`-`000015` files remain available for historical traceability.
- The squashed `000005_v10_productization_bundle` file was validated against this archived chain for schema equivalence and rollback behavior.
