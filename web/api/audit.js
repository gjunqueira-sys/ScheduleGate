/**
 * Vercel serverless function — free-audit intake form POST handler.
 * Forwards the request to support@schedulegate.dev via the Resend HTTP API.
 *
 * Requires one Vercel project environment variable:
 *   RESEND_AUDIT_KEY = a Resend API key valid for the schedulegate.dev domain
 *
 * If the env var is missing, the endpoint returns { ok:false, notConfigured:true }
 * and the page falls back to a pre-filled mailto compose, so /audit still works
 * with zero configuration.
 */

export default function handler(req, res) {
  if (req.method !== "POST") {
    return res.status(405).json({ error: "Method not allowed" });
  }

  const key = process.env.RESEND_AUDIT_KEY;
  if (!key) {
    return res.json({ ok: false, notConfigured: true });
  }

  let body = "";
  req.on("data", (chunk) => {
    body += chunk;
  });

  req.on("end", async () => {
    let data;
    try {
      data = JSON.parse(body || "{}");
    } catch {
      return res.status(400).json({ ok: false, error: "bad_json" });
    }

    const email = String(data.email || "").trim().toLowerCase();
    const name = String(data.name || "").trim();
    if (!email || !name || !email.includes("@")) {
      return res.status(400).json({ ok: false, error: "missing_fields" });
    }

    const text = [
      `Name: ${name}`,
      `Email: ${email}`,
      `Role: ${String(data.role || "").trim() || "-"}`,
      `Source: ${String(data.source || "").trim() || "-"}`,
      `Activity count: ${String(data.size || "").trim() || "-"}`,
      `Purpose: ${String(data.message || "").trim() || "-"}`,
      "",
      "Source: https://schedulegate.dev/audit",
      "Action: reply within one business day and arrange delivery of the",
      "sanitised export. Delete the source file within 7 days after delivery.",
    ].join("\n");

    try {
      const r = await fetch("https://api.resend.com/emails", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${key}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          from: "ScheduleGate <noreply@schedulegate.dev>",
          to: ["support@schedulegate.dev"],
          reply_to: email,
          subject: `Free DCMA audit request — ${name}`,
          text,
        }),
      });

      if (r.ok) {
        return res.json({ ok: true });
      }
      const errText = await r.text().catch(() => "");
      console.error("Resend delivery failed:", r.status, errText);
      return res.status(502).json({ ok: false, error: "provider_error" });
    } catch (err) {
      console.error("Audit form handler error:", err);
      return res.status(500).json({ ok: false, error: "internal" });
    }
  });
}