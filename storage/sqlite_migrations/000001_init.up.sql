create table if not exists events(
    eventid text primary key,
    eventtype text not null,
    eventcreatedat int not null,
    eventheaders json not null,
    eventdata json not null
);
