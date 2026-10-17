-- +migrate Up

-- the unique pair also finds an idea's me toos when the garden counts
-- them
create table me_toos
(
    id      uuid    not null primary key,
    version integer not null,
    idea_id uuid    not null references ideas (id),
    user_id uuid    not null references users (id),
    unique (idea_id, user_id)
);

-- +migrate Down

drop table me_toos;
