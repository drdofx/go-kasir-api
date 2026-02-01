create table if not exists products (
  id bigserial primary key,
  name varchar not null,
  price int4 not null,
  stock int4 not null,
  category_id bigint
);

do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'products_category_id_fkey'
  ) then
    alter table products
    add constraint products_category_id_fkey
    foreign key (category_id) references categories(id)
    on delete set null;
  end if;
end $$;

create index if not exists products_category_id_idx on products(category_id);
