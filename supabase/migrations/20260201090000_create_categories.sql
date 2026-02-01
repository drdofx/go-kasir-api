create table if not exists categories (
  id bigserial primary key,
  name varchar not null,
  description text not null
);
