-- +migrate Up

-- every member of a project is equal, and every role stored so far was
-- the owner's, so dropping it loses nothing
alter table memberships drop column role;

-- +migrate Down

alter table memberships add column role text not null default 'owner';
