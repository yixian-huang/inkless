import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import type { SectionProps } from "../types";
import { useFormSubmit } from "@/hooks/useFormSubmit";

export interface ContactFormData {
  title?: string;
  subtitle?: string;
  nameLabel?: string;
  namePlaceholder?: string;
  emailLabel?: string;
  emailPlaceholder?: string;
  messageLabel?: string;
  messagePlaceholder?: string;
  submit?: string;
  phone?: string;
  address?: string;
  accentColor?: string;
}

export default function ContactFormSection({ data }: SectionProps<ContactFormData>) {
  const {
    title,
    nameLabel,
    namePlaceholder,
    emailLabel,
    emailPlaceholder,
    messageLabel,
    messagePlaceholder,
    submit,
    phone,
    address,
    accentColor,
  } = data;

  const { i18n } = useTranslation();
  const locale = i18n.language?.startsWith("zh") ? "zh" : "en";
  const { submit: submitFormData, isSubmitting, isSuccess, error: submitError, reset: resetSubmit } = useFormSubmit({
    formType: "contact",
    locale,
  });

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await submitFormData({ name, email, message });
  };

  useEffect(() => {
    if (isSuccess) {
      setName("");
      setEmail("");
      setMessage("");
    }
  }, [isSuccess]);

  const focusRingClass = accentColor
    ? undefined
    : "focus:ring-primary";

  return (
    <div className="max-w-layout mx-auto px-4 md:px-content xl:px-8">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-16 items-start">
        <div>
          {title && (
            <div className="flex items-center mb-2">
              <div className="w-[26px] h-[26px] bg-accent mr-3 flex-shrink-0 rounded-full" />
              <h2 className="text-xl md:text-2xl font-bold text-primary">
                {title}
              </h2>
            </div>
          )}
        </div>
        <div className="space-y-6">
          {phone && (
            <div className="flex items-start gap-4">
              <span className="text-primary mt-1 flex-shrink-0" aria-hidden>
                <svg
                  className="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z"
                  />
                </svg>
              </span>
              <div>
                <p className="text-on-surface font-medium">{phone}</p>
              </div>
            </div>
          )}
          {address && (
            <div className="flex items-start gap-4">
              <span className="text-primary mt-1 flex-shrink-0" aria-hidden>
                <svg
                  className="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
                  />
                </svg>
              </span>
              <div>
                <p className="text-on-surface">{address}</p>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="flex flex-col items-center mt-10 md:mt-14">
        {isSuccess && (
          <div className="w-full max-w-md mb-4 p-4 bg-green-50 border border-green-200 rounded-md text-green-700 text-sm flex items-center justify-between">
            <span>{locale === "zh" ? "提交成功！我们会尽快与您联系。" : "Submitted successfully! We will contact you soon."}</span>
            <button onClick={resetSubmit} className="ml-2 text-green-500 hover:text-green-700">&times;</button>
          </div>
        )}
        {submitError && (
          <div className="w-full max-w-md mb-4 p-4 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm flex items-center justify-between">
            <span>{submitError}</span>
            <button onClick={resetSubmit} className="ml-2 text-red-500 hover:text-red-700">&times;</button>
          </div>
        )}
        <form onSubmit={handleSubmit} className="w-full max-w-md space-y-4">
          <div>
            <label
              htmlFor="section-contact-name"
              className="block text-on-surface text-sm font-medium mb-1"
            >
              {nameLabel || (locale === "zh" ? "姓名" : "Name")} *
            </label>
            <input
              id="section-contact-name"
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={namePlaceholder}
              className={`w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:border-transparent ${focusRingClass || ""}`}
              style={accentColor ? { "--tw-ring-color": accentColor } as React.CSSProperties : undefined}
            />
          </div>
          <div>
            <label
              htmlFor="section-contact-email"
              className="block text-on-surface text-sm font-medium mb-1"
            >
              {emailLabel || (locale === "zh" ? "邮箱" : "Email")} *
            </label>
            <input
              id="section-contact-email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={emailPlaceholder}
              className={`w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:border-transparent ${focusRingClass || ""}`}
              style={accentColor ? { "--tw-ring-color": accentColor } as React.CSSProperties : undefined}
            />
          </div>
          <div>
            <label
              htmlFor="section-contact-message"
              className="block text-on-surface text-sm font-medium mb-1"
            >
              {messageLabel || (locale === "zh" ? "留言" : "Message")}
            </label>
            <textarea
              id="section-contact-message"
              rows={5}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder={messagePlaceholder}
              className={`w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:border-transparent resize-y ${focusRingClass || ""}`}
              style={accentColor ? { "--tw-ring-color": accentColor } as React.CSSProperties : undefined}
            />
          </div>
          <div className="flex justify-center">
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-8 py-3 rounded-md text-white font-medium transition-colors cursor-pointer bg-primary hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
              style={accentColor ? { backgroundColor: accentColor } : undefined}
            >
              {isSubmitting ? (locale === "zh" ? "提交中..." : "Submitting...") : (submit || "Submit")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
