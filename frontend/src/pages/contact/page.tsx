import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import PageHero from '../../components/feature/PageHero';
import { usePublicContent } from '@/hooks/usePublicContent';
import { useFormSubmit } from '@/hooks/useFormSubmit';
import { resolveLocale } from '@/utils/locale';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';

interface HeroConfig {
  title?: string;
  subtitle?: string;
  backgroundColor?: string;
}

interface FormConfig {
  title?: string;
  subtitle?: string;
  nameLabel?: string;
  namePlaceholder?: string;
  emailLabel?: string;
  emailPlaceholder?: string;
  messageLabel?: string;
  messagePlaceholder?: string;
  submitLabel?: string;
}

interface ContactInfo {
  phone?: string;
  email?: string;
  address?: string;
}

interface ContactPageConfig {
  hero?: HeroConfig;
  form?: FormConfig;
  contactInfo?: ContactInfo;
}

export default function ContactPage() {
  useDocumentTitle("联系我们");
  const { i18n } = useTranslation('common');
  const locale = resolveLocale(i18n.language);

  const { loading, error, config } = usePublicContent('contact', {
    locale,
    autoNormalize: true,
  });

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [message, setMessage] = useState('');

  const { submit, isSubmitting, isSuccess, error: submitError, reset: resetSubmit } = useFormSubmit({
    formType: "contact",
    locale,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await submit({ name, email, message });
  };

  useEffect(() => {
    if (isSuccess) {
      setName("");
      setEmail("");
      setMessage("");
    }
  }, [isSuccess]);

  if (loading) {
    return (
      <>
        <div className="min-h-screen bg-white flex items-center justify-center">
          <div className="text-gray-600">Loading...</div>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <div className="min-h-screen bg-white flex items-center justify-center">
          <div className="text-red-600">Failed to load page content</div>
        </div>
      </>
    );
  }

  const pageConfig = (config as ContactPageConfig) || {};
  const hero = pageConfig.hero || {};
  const form = pageConfig.form || {};
  const contact = pageConfig.contactInfo || {};
  const heroBgColor = hero.backgroundColor || '#1E9188';

  return (
    <>
      <PageHero
        title={hero.title}
        backgroundColor={heroBgColor}
      />

      {/* 主内容：标题+联系方式左右布局，表单区域单独居中 */}
      <section className="py-12 md:py-16 lg:py-24 bg-white">
        <div className="max-w-layout mx-auto px-4 md:px-6">
          {/* 左右布局：联络我们的专家+副标题 | 联系方式 */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 lg:gap-16 items-start">
            <div>
              {form.title && (
                <div className="flex items-center mb-2">
                  <div className="w-[26px] h-[26px] bg-accent mr-3 flex-shrink-0 rounded-full" />
                  <h2 className="text-xl md:text-2xl font-bold text-primary">
                    {form.title}
                  </h2>
                </div>
              )}
            </div>
            <div className="space-y-6">
              {contact.phone && (
                <div className="flex items-start gap-4">
                  <span className="text-primary mt-1 flex-shrink-0" aria-hidden>
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
                    </svg>
                  </span>
                  <div>
                    <p className="text-gray-900 font-medium">{contact.phone}</p>
                  </div>
                </div>
              )}
              {contact.email && (
                <div className="flex items-start gap-4">
                  <span className="text-primary mt-1 flex-shrink-0" aria-hidden>
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                  </span>
                  <div>
                    <p className="text-gray-900 font-medium">{contact.email}</p>
                  </div>
                </div>
              )}
              {contact.address && (
                <div className="flex items-start gap-4">
                  <span className="text-primary mt-1 flex-shrink-0" aria-hidden>
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </span>
                  <div>
                    <p className="text-gray-900">{contact.address}</p>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* 表单区域单独一行，相对页面水平居中 */}
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
                <label htmlFor="contact-name" className="block text-gray-900 text-sm font-medium mb-1">
                  {form.nameLabel || (locale === "zh" ? "姓名" : "Name")} *
                </label>
                <input
                  id="contact-name"
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={form.namePlaceholder || (locale === "zh" ? "请输入您的姓名" : "Your name")}
                  className="w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                />
              </div>
              <div>
                <label htmlFor="contact-email" className="block text-gray-900 text-sm font-medium mb-1">
                  {form.emailLabel || (locale === "zh" ? "邮箱" : "Email")} *
                </label>
                <input
                  id="contact-email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={form.emailPlaceholder || (locale === "zh" ? "请输入您的邮箱地址" : "Your email address")}
                  className="w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                />
              </div>
              <div>
                <label htmlFor="contact-message" className="block text-gray-900 text-sm font-medium mb-1">
                  {form.messageLabel || (locale === "zh" ? "留言" : "Message")}
                </label>
                <textarea
                  id="contact-message"
                  rows={5}
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder={form.messagePlaceholder || (locale === "zh" ? "请输入您的留言" : "Your message")}
                  className="w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent resize-y"
                />
              </div>
              <div className="flex justify-center">
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="px-8 py-3 rounded-md text-white font-medium transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{ backgroundColor: heroBgColor }}
                >
                  {isSubmitting ? (locale === "zh" ? "提交中..." : "Submitting...") : (form.submitLabel || "Submit")}
                </button>
              </div>
            </form>
          </div>
        </div>
      </section>
    </>
  );
}
