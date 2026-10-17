-- +migrate Up

-- every public project so far could only be joined through a member, and
-- a private project is always closed, since nobody outside can see it
alter table projects
    add column joining text not null default 'closed'
        check (joining in ('open', 'closed') and (joining = 'closed' or visibility = 'public'));

-- +migrate Down

alter table projects drop column joining;
