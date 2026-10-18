-- +migrate Up

alter table projects drop constraint projects_joining_check;
alter table projects
    add constraint projects_joining_check check (joining in ('open', 'request', 'closed'));

-- +migrate Down

-- a project that took requests becomes closed, not open, so nobody gets
-- in without a member

-- the version trigger only lets an update through that writes the next
-- version
update projects set joining = 'closed', version = version + 1 where joining = 'request';

alter table projects drop constraint projects_joining_check;
alter table projects
    add constraint projects_joining_check check (joining in ('open', 'closed'));
