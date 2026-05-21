import type { EmailConfig } from "./types";

const autoReplyZhBody = `<!DOCTYPE html>
<html lang="zh">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f7;padding:40px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
        <tr><td style="background:linear-gradient(135deg,#1e40af,#3b82f6);padding:32px 40px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">{{siteName}}</h1>
          <p style="margin:8px 0 0;color:rgba(255,255,255,0.85);font-size:14px;">{{siteNameEn}}</p>
        </td></tr>
        <tr><td style="padding:32px 40px;">
          <p style="margin:0 0 16px;color:#374151;font-size:16px;line-height:1.6;">尊敬的 {{name}}，您好！</p>
          <p style="margin:0 0 16px;color:#374151;font-size:15px;line-height:1.6;">感谢您通过我们的网站提交咨询。我们已收到您的留言，团队将在 <strong>1-2 个工作日</strong> 内与您联系。</p>
          <div style="background:#f0f9ff;border-left:4px solid #3b82f6;padding:16px 20px;margin:24px 0;border-radius:0 6px 6px 0;">
            <p style="margin:0;color:#1e40af;font-size:14px;">如有紧急事项，请直接拨打我们的联系电话。</p>
          </div>
          <p style="margin:24px 0 0;color:#6b7280;font-size:14px;line-height:1.6;">此致<br/>{{siteName}}团队</p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:20px 40px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;color:#9ca3af;font-size:12px;text-align:center;">此邮件由系统自动发送，请勿直接回复。</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`;

const autoReplyEnBody = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f7;padding:40px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
        <tr><td style="background:linear-gradient(135deg,#1e40af,#3b82f6);padding:32px 40px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">{{siteNameEn}}</h1>
          <p style="margin:8px 0 0;color:rgba(255,255,255,0.85);font-size:14px;">Professional Consulting Services</p>
        </td></tr>
        <tr><td style="padding:32px 40px;">
          <p style="margin:0 0 16px;color:#374151;font-size:16px;line-height:1.6;">Dear {{name}},</p>
          <p style="margin:0 0 16px;color:#374151;font-size:15px;line-height:1.6;">Thank you for reaching out through our website. We have received your inquiry and our team will get back to you within <strong>1-2 business days</strong>.</p>
          <div style="background:#f0f9ff;border-left:4px solid #3b82f6;padding:16px 20px;margin:24px 0;border-radius:0 6px 6px 0;">
            <p style="margin:0;color:#1e40af;font-size:14px;">For urgent matters, please contact us by phone directly.</p>
          </div>
          <p style="margin:24px 0 0;color:#6b7280;font-size:14px;line-height:1.6;">Best regards,<br/>{{siteNameEn}} Team</p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:20px 40px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;color:#9ca3af;font-size:12px;text-align:center;">This is an automated message. Please do not reply directly.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`;

const forwardZhBody = `<!DOCTYPE html>
<html lang="zh">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f7;padding:40px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
        <tr><td style="background:linear-gradient(135deg,#7c3aed,#a855f7);padding:32px 40px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">新表单提交通知</h1>
          <p style="margin:8px 0 0;color:rgba(255,255,255,0.85);font-size:14px;">{{siteName}} - 官网留言</p>
        </td></tr>
        <tr><td style="padding:32px 40px;">
          <p style="margin:0 0 20px;color:#374151;font-size:15px;">收到一条新的网站咨询，详情如下：</p>
          <table width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e5e7eb;border-radius:8px;overflow:hidden;">
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;width:100px;border-bottom:1px solid #e5e7eb;">姓名</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{name}}</td>
            </tr>
            <tr>
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">邮箱</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{email}}</td>
            </tr>
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">电话</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{phone}}</td>
            </tr>
            <tr>
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">公司</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{company}}</td>
            </tr>
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;vertical-align:top;">留言</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;line-height:1.6;">{{message}}</td>
            </tr>
          </table>
          <p style="margin:20px 0 0;color:#9ca3af;font-size:12px;">提交时间：{{date}}</p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:20px 40px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;color:#9ca3af;font-size:12px;text-align:center;">此邮件由系统自动发送，请勿直接回复。</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`;

const forwardEnBody = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f5f7;padding:40px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
        <tr><td style="background:linear-gradient(135deg,#7c3aed,#a855f7);padding:32px 40px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:600;">New Form Submission</h1>
          <p style="margin:8px 0 0;color:rgba(255,255,255,0.85);font-size:14px;">{{siteNameEn}} - Website Inquiry</p>
        </td></tr>
        <tr><td style="padding:32px 40px;">
          <p style="margin:0 0 20px;color:#374151;font-size:15px;">A new inquiry has been submitted via the website:</p>
          <table width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e5e7eb;border-radius:8px;overflow:hidden;">
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;width:100px;border-bottom:1px solid #e5e7eb;">Name</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{name}}</td>
            </tr>
            <tr>
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">Email</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{email}}</td>
            </tr>
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">Phone</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{phone}}</td>
            </tr>
            <tr>
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;border-bottom:1px solid #e5e7eb;">Company</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;border-bottom:1px solid #e5e7eb;">{{company}}</td>
            </tr>
            <tr style="background:#f9fafb;">
              <td style="padding:12px 16px;font-size:13px;color:#6b7280;font-weight:600;vertical-align:top;">Message</td>
              <td style="padding:12px 16px;font-size:14px;color:#111827;line-height:1.6;">{{message}}</td>
            </tr>
          </table>
          <p style="margin:20px 0 0;color:#9ca3af;font-size:12px;">Submitted at: {{date}}</p>
        </td></tr>
        <tr><td style="background:#f9fafb;padding:20px 40px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;color:#9ca3af;font-size:12px;text-align:center;">This is an automated message. Please do not reply directly.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`;

export const defaultEmailConfig: EmailConfig = {
  smtp: {
    host: "",
    port: 587,
    username: "",
    password: "",
    from: "",
    fromName: "",
    useTLS: true,
    insecureSkipVerify: false,
  },
  receiver: {
    enabled: true,
    emails: "",
  },
  autoReply: {
    enabled: true,
  },
  templates: {
    autoReply: {
      zh: {
        subject: "感谢您的咨询",
        body: autoReplyZhBody,
      },
      en: {
        subject: "Thank you for your inquiry",
        body: autoReplyEnBody,
      },
    },
    forward: {
      zh: {
        subject: "【官网留言】来自 {{name}} 的新咨询",
        body: forwardZhBody,
      },
      en: {
        subject: "[Website Inquiry] New submission from {{name}}",
        body: forwardEnBody,
      },
    },
  },
};
