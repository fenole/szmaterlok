insert into events
    ( eventid
    , eventtype
    , eventcreatedat
    , eventheaders
    , eventdata )
values
    ( :id
    , :type
    , :createdat
    , :headers
    , :data );
