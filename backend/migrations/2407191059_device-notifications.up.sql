CREATE TABLE device_notification_token (
    device_id integer references device(id) ON DELETE CASCADE PRIMARY KEY,
    notification_token VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

alter table device drop column last_seen;

create table device_call_outbox (
    id serial primary key,
    device_id integer references device(id) ON DELETE CASCADE,
    call_id varchar(255) unique not null,
    jitsi_room_id varchar(255) not null,
    jitsi_jwt TEXT not null,
    created_at timestamp not null default now()
);