-- +migrate Up

-- A project nobody is in turns public and open from now on, when its last
-- member leaves. Projects that were left empty before get the same once,
-- so none stays lost. Their requests go, since nobody can answer them.

delete from join_requests
where not exists (select 1 from memberships m where m.project_id = join_requests.project_id);

-- the version trigger only lets an update through that writes the next
-- version
update projects
set visibility = 'public',
    joining    = 'open',
    version    = version + 1
where not exists (select 1 from memberships m where m.project_id = projects.id)
  and not (visibility = 'public' and joining = 'open');

-- +migrate Down

-- Nothing to undo. Which projects were private or closed before isn't
-- kept, and an open project nobody is in is what frogly makes anyway.
