import { useState, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { ADMIN_DEFAULT_PATH } from "@/router/adminAccess";
import { PRODUCT_BRAND } from "@/config/productBrand";
import { AdminButton, AdminInput, AdminErrorBanner } from "@/components/admin/ui";

export default function LoginPage() {
  useDocumentTitle(`${PRODUCT_BRAND.name} 登录`);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { login } = useAuth();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      await login(username, password);
      navigate(ADMIN_DEFAULT_PATH);
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败，请重试");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="admin-scope relative flex min-h-screen items-center justify-center overflow-hidden bg-[#f5f1ea] px-4 py-12 sm:px-6">
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage:
            "radial-gradient(ellipse 70% 50% at 50% 0%, rgba(26,24,20,0.06), transparent 55%), radial-gradient(ellipse 50% 40% at 100% 100%, rgba(155,59,46,0.04), transparent 50%)",
        }}
        aria-hidden
      />

      <div className="relative w-full max-w-[420px]">
        <div className="mb-8 text-center">
          <div className="mb-5 flex justify-center">
            <div className="rounded-xl border border-[#e4ddd2] bg-[#fbfaf7] p-3 shadow-[0_8px_28px_rgba(26,24,20,0.06)]">
              <img className="h-10 w-10" src="/brand/inkless-mark-ink.svg" alt="Inkless" />
            </div>
          </div>
          <h1 className="text-2xl font-semibold tracking-[-0.02em] text-[#1a1814]">
            {PRODUCT_BRAND.name} 管理后台
          </h1>
          <p className="mt-2 text-sm tracking-wide text-[#8a8378]">使用管理员账号登录以继续</p>
        </div>

        <div className="rounded-xl border border-[#e4ddd2] bg-[#fbfaf7] p-6 shadow-[0_12px_40px_rgba(26,24,20,0.06)] sm:p-8">
          <form className="space-y-5" onSubmit={handleSubmit}>
            {error ? (
              <AdminErrorBanner message={error} className="mb-0" onDismiss={() => setError("")} />
            ) : null}

            <div>
              <label htmlFor="username" className="mb-1.5 block text-sm font-medium text-[#3d3832]">
                用户名
              </label>
              <AdminInput
                id="username"
                name="username"
                type="text"
                autoComplete="username"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="输入用户名"
              />
            </div>

            <div>
              <label htmlFor="password" className="mb-1.5 block text-sm font-medium text-[#3d3832]">
                密码
              </label>
              <AdminInput
                id="password"
                name="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="输入密码"
              />
            </div>

            <AdminButton type="submit" disabled={loading} size="lg" className="w-full">
              {loading ? "登录中…" : "登录"}
            </AdminButton>
          </form>
        </div>

        <p className="mt-6 text-center text-xs tracking-wide text-[#a39b90]">
          {PRODUCT_BRAND.fullName} · {PRODUCT_BRAND.domain}
        </p>
      </div>
    </div>
  );
}
