import { useState, type FormEvent } from "react";
import { EfShell } from "./shell";
import { asBool, asString, type SectionProps } from "./types";

export interface EfContactSplitData {
  title?: string;
  intro?: string;
  phone?: string;
  email?: string;
  address?: string;
  showForm?: boolean;
  nameLabel?: string;
  emailLabel?: string;
  messageLabel?: string;
  submitLabel?: string;
}

/**
 * POST to host public form-submissions endpoint (same path as app http client).
 * On failure, surface mailto fallback using the contact email prop.
 */
async function submitContact(payload: {
  name: string;
  email: string;
  message: string;
}): Promise<void> {
  const res = await fetch("/public/form-submissions", {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({
      formType: "contact",
      name: payload.name,
      email: payload.email,
      message: payload.message,
      sourceUrl: typeof window !== "undefined" ? window.location.href : undefined,
    }),
  });
  if (!res.ok) {
    let message = `Submission failed (${res.status})`;
    try {
      const body = await res.json();
      message = body?.error?.message || body?.message || message;
    } catch {
      /* ignore parse errors */
    }
    throw new Error(message);
  }
}

export default function EfContactSplit({ data }: SectionProps<EfContactSplitData>) {
  const title = asString(data.title);
  const intro = asString(data.intro);
  const phone = asString(data.phone);
  const email = asString(data.email);
  const address = asString(data.address);
  const showForm = asBool(data.showForm, true);
  const nameLabel = asString(data.nameLabel, "Name") || "Name";
  const emailLabel = asString(data.emailLabel, "Email") || "Email";
  const messageLabel = asString(data.messageLabel, "Message") || "Message";
  const submitLabel = asString(data.submitLabel, "Send") || "Send";

  const [name, setName] = useState("");
  const [formEmail, setFormEmail] = useState("");
  const [message, setMessage] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showMailto, setShowMailto] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    setShowMailto(false);
    setIsSuccess(false);
    try {
      await submitContact({ name, email: formEmail, message });
      setIsSuccess(true);
      setName("");
      setFormEmail("");
      setMessage("");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Submission failed";
      setError(msg);
      setShowMailto(true);
    } finally {
      setIsSubmitting(false);
    }
  };

  const mailtoHref =
    email
      ? `mailto:${email}?subject=${encodeURIComponent("Contact")}&body=${encodeURIComponent(
          `Name: ${name}\nEmail: ${formEmail}\n\n${message}`,
        )}`
      : "";

  return (
    <EfShell>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 lg:gap-16 items-start">
        <div>
          {title ? (
            <h2 className="font-heading text-3xl md:text-4xl text-on-surface font-semibold mb-4">
              {title}
            </h2>
          ) : null}
          {intro ? (
            <p className="text-base md:text-lg text-on-surface-muted leading-relaxed mb-8">
              {intro}
            </p>
          ) : null}

          <dl className="space-y-4 text-sm md:text-base">
            {phone ? (
              <div>
                <dt className="text-xs uppercase tracking-wider text-on-surface-muted mb-1">
                  Phone
                </dt>
                <dd className="text-on-surface">
                  <a href={`tel:${phone.replace(/\s+/g, "")}`} className="hover:text-accent transition-colors">
                    {phone}
                  </a>
                </dd>
              </div>
            ) : null}
            {email ? (
              <div>
                <dt className="text-xs uppercase tracking-wider text-on-surface-muted mb-1">
                  Email
                </dt>
                <dd className="text-on-surface">
                  <a href={`mailto:${email}`} className="hover:text-accent transition-colors">
                    {email}
                  </a>
                </dd>
              </div>
            ) : null}
            {address ? (
              <div>
                <dt className="text-xs uppercase tracking-wider text-on-surface-muted mb-1">
                  Address
                </dt>
                <dd className="text-on-surface whitespace-pre-line">{address}</dd>
              </div>
            ) : null}
          </dl>
        </div>

        {showForm ? (
          <div>
            {isSuccess ? (
              <p className="mb-4 p-4 border border-border bg-surface-alt text-on-surface text-sm">
                Thank you — we will be in touch soon.
              </p>
            ) : null}
            {error ? (
              <div className="mb-4 p-4 border border-border bg-surface-alt text-sm text-on-surface space-y-2">
                <p>{error}</p>
                {showMailto && mailtoHref ? (
                  <p>
                    <a href={mailtoHref} className="text-accent underline underline-offset-2">
                      Or email us directly
                    </a>
                  </p>
                ) : null}
              </div>
            ) : null}

            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label htmlFor="ef-contact-name" className="block text-sm text-on-surface mb-1.5">
                  {nameLabel} *
                </label>
                <input
                  id="ef-contact-name"
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-4 py-3 bg-surface border border-border rounded-sm text-on-surface focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent"
                />
              </div>
              <div>
                <label htmlFor="ef-contact-email" className="block text-sm text-on-surface mb-1.5">
                  {emailLabel} *
                </label>
                <input
                  id="ef-contact-email"
                  type="email"
                  required
                  value={formEmail}
                  onChange={(e) => setFormEmail(e.target.value)}
                  className="w-full px-4 py-3 bg-surface border border-border rounded-sm text-on-surface focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent"
                />
              </div>
              <div>
                <label htmlFor="ef-contact-message" className="block text-sm text-on-surface mb-1.5">
                  {messageLabel}
                </label>
                <textarea
                  id="ef-contact-message"
                  rows={5}
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  className="w-full px-4 py-3 bg-surface border border-border rounded-sm text-on-surface focus:outline-none focus:ring-1 focus:ring-accent focus:border-accent resize-y"
                />
              </div>
              <button
                type="submit"
                disabled={isSubmitting}
                className="px-7 py-3 bg-primary text-on-primary text-sm uppercase tracking-wider font-medium hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isSubmitting ? "…" : submitLabel}
              </button>
            </form>
          </div>
        ) : null}
      </div>
    </EfShell>
  );
}
