# The data layer

frogly stores its data in SQLite or Postgres, chosen with
`FROGLY_DATABASE__DRIVER`. The rest of frogly doesn't know which one it
talks to.

- `internal/repositories` holds the models, like `User` and `Project`,
  and the interfaces of their repositories. A model's fields are private.
  Its constructor, like `NewProject`, checks the values and refuses what
  frogly can't store. A second function, like `NewProjectFromDB`, rebuilds
  a model from the database without checking.
- `internal/db` holds the unit of work, `db.Context`. Each request gets one
  of its own. Reads through its repositories go to the database right
  away. Writes like `Insert` and `Update` are only recorded, and
  `SaveChanges` writes them all in one transaction. An update writes the
  whole row.
- Each database has its own adapter, in `internal/db/sqlite` and
  `internal/repositories/sqlite`, and the same for Postgres. Each adapter
  has its own SQL, even where the two look alike.
- Migrations live in `internal/db/<database>/migrations`, numbered, one
  folder per database. frogly runs them when it starts.
- A row has a version. A trigger in the schema refuses an update that
  doesn't write the next one, so a change can't overwrite one that was
  saved in between. The adapter reports that as
  `repositories.ErrConcurrentModification`.
- `internal/db/dbtest` is the contract every adapter meets. Both adapters
  run it, Postgres with the `integration` tag.

To add a model:

1. First, add the model and its repository interface to
   `internal/repositories`, with tests for its constructor.
2. Then add the repository to `db.Context`, and the contract's tests for it
   to `internal/db/dbtest`.
3. Add a migration and the repository to each adapter, until both pass
   the contract. A model that can be updated needs the version trigger in
   both migrations: in Postgres, a trigger that executes the existing
   `check_version()` function, in SQLite, a trigger like
   `users_version_check` that raises an error when the new version isn't
   the old one plus one.
4. Finally, add a rule for any new package to `arch-go.yml`.
