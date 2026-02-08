create table if not exists transactions (
  id bigserial primary key,
  total_amount int not null,
  created_at timestamp default current_timestamp
);

create table if not exists transaction_details (
  id bigserial primary key,
  transaction_id int references transactions(id) on delete cascade,
  product_id int references products(id),
  quantity int not null,
  subtotal int not null
);

create index if not exists transaction_details_transaction_id_idx on transaction_details(transaction_id);
create index if not exists transaction_details_product_id_idx on transaction_details(product_id);
