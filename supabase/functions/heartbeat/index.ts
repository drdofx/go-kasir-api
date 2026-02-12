import { createClient } from "https://esm.sh/@supabase/supabase-js@2.90.1";

console.info("server started");

Deno.serve(async () => {
  const url = Deno.env.get("SUPABASE_URL");
  const key = Deno.env.get("SUPABASE_ANON_KEY");

  if (!url || !key) {
    return new Response(JSON.stringify({ ok: false, error: "Missing env" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }

  const supabase = createClient(url, key);
  const { error } = await supabase.rpc("heartbeat_rpc");

  return new Response(
    JSON.stringify({ ok: !error, error: error?.message ?? null }),
    {
      status: error ? 500 : 200,
      headers: { "Content-Type": "application/json" },
    },
  );
});
