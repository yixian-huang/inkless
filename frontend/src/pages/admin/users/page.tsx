import { useState } from "react";
import {
  listUsers,
  createUser,
  updateUser,
  deleteUser,
  type UserDTO,
  type CreateUserRequest,
  type UpdateUserRequest,
} from "@/api/users";
import {
  AdminBadge,
  AdminButton,
  AdminErrorBanner,
  AdminLoading,
  AdminPageHeader,
  AdminPagination,
  AdminTable,
  AdminTableBody,
  AdminTableHead,
  AdminTd,
  AdminTh,
  useAdminConfirm,
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { invalidateAdminQueryPrefix, useAdminQuery } from "@/lib/adminQuery";
import { adminQueryKeys } from "@/lib/adminQueryKeys";

const ALL_PERMISSIONS = [
  { key: "dashboard", label: "仪表盘" },
  { key: "content", label: "内容管理" },
  { key: "pages", label: "页面管理" },
  { key: "articles", label: "文章管理" },
  { key: "media", label: "媒体管理" },
  { key: "form-submissions", label: "表单提交" },
  { key: "menus", label: "菜单管理" },
  { key: "theme", label: "主题" },
  { key: "analytics", label: "访问统计" },
  { key: "audit-logs", label: "审计日志" },
  { key: "backups", label: "数据备份" },
  { key: "users", label: "用户管理" },
];

interface UserFormData {
  username: string;
  password: string;
  role: string;
  permissions: string[];
}

const emptyForm: UserFormData = {
  username: "",
  password: "",
  role: "editor",
  permissions: [],
};

export default function AdminUsersPage() {
  useDocumentTitle("用户管理");
  const [page, setPage] = useState(1);

  // Dialog state
  const [showDialog, setShowDialog] = useState(false);
  const [editingUser, setEditingUser] = useState<UserDTO | null>(null);
  const [form, setForm] = useState<UserFormData>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");

  const { confirm, confirmDialog } = useAdminConfirm();
  const pageSize = 20;

  const { data, error, loading, refetch } = useAdminQuery(
    [...adminQueryKeys.users, page, pageSize],
    () => listUsers(page, pageSize),
  );
  const users = data?.items ?? [];
  const total = data?.total ?? 0;

  const fetchUsers = async () => {
    invalidateAdminQueryPrefix(adminQueryKeys.users);
    await refetch({ force: true });
  };

  const openCreate = () => {
    setEditingUser(null);
    setForm(emptyForm);
    setFormError("");
    setShowDialog(true);
  };

  const openEdit = (user: UserDTO) => {
    setEditingUser(user);
    setForm({
      username: user.username,
      password: "",
      role: user.role,
      permissions: [...user.permissions],
    });
    setFormError("");
    setShowDialog(true);
  };

  const handleSave = async () => {
    setFormError("");
    if (!form.username.trim()) {
      setFormError("请输入用户名");
      return;
    }
    if (!editingUser && form.password.length < 6) {
      setFormError("密码长度不能少于6位");
      return;
    }
    if (editingUser && form.password && form.password.length < 6) {
      setFormError("密码长度不能少于6位");
      return;
    }

    setSaving(true);
    try {
      if (editingUser) {
        const data: UpdateUserRequest = {
          username: form.username,
          role: form.role,
          permissions: form.permissions,
        };
        if (form.password) {
          data.password = form.password;
        }
        await updateUser(editingUser.id, data);
      } else {
        const data: CreateUserRequest = {
          username: form.username,
          password: form.password,
          role: form.role,
          permissions: form.permissions,
        };
        await createUser(data);
      }
      setShowDialog(false);
      fetchUsers();
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message || "保存失败";
      setFormError(msg);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (user: UserDTO) => {
    const ok = await confirm({
      title: "删除用户",
      message: `确定删除用户「${user.username}」吗？此操作不可撤销。`,
      confirmLabel: "删除",
      danger: true,
    });
    if (!ok) return;
    try {
      await deleteUser(user.id);
      await fetchUsers();
    } catch (err: any) {
      const msg = err?.response?.data?.error?.message || "删除失败";
      alert(msg);
    }
  };

  const togglePermission = (key: string) => {
    setForm((prev) => ({
      ...prev,
      permissions: prev.permissions.includes(key)
        ? prev.permissions.filter((p) => p !== key)
        : [...prev.permissions, key],
    }));
  };

  const toggleAllPermissions = () => {
    const allKeys = ALL_PERMISSIONS.map((p) => p.key);
    const allSelected = allKeys.every((k) => form.permissions.includes(k));
    setForm((prev) => ({
      ...prev,
      permissions: allSelected ? [] : [...allKeys],
    }));
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="space-y-6">
      {confirmDialog}
      <AdminPageHeader
        title="用户管理"
        description="管理后台账号与基础权限"
        actions={
          <AdminButton size="sm" onClick={openCreate}>
            创建用户
          </AdminButton>
        }
      />

      {error && <AdminErrorBanner message={error.message || "加载用户列表失败"} />}

      {loading ? (
        <AdminLoading />
      ) : (
        <>
          <AdminTable>
            <AdminTableHead>
              <tr>
                <AdminTh>用户名</AdminTh>
                <AdminTh>角色</AdminTh>
                <AdminTh>超管</AdminTh>
                <AdminTh>权限数</AdminTh>
                <AdminTh>创建时间</AdminTh>
                <AdminTh className="text-right">操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {users.length === 0 ? (
                <tr>
                  <AdminTd colSpan={6} className="py-8 text-center text-slate-500">
                    暂无用户
                  </AdminTd>
                </tr>
              ) : (
                users.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-50/80">
                    <AdminTd className="font-medium text-slate-900">{u.username}</AdminTd>
                    <AdminTd>{u.role === "admin" ? "管理员" : "编辑"}</AdminTd>
                    <AdminTd>
                      {u.isSuperAdmin ? (
                        <AdminBadge tone="warning">超级管理员</AdminBadge>
                      ) : (
                        <span className="text-slate-400">—</span>
                      )}
                    </AdminTd>
                    <AdminTd className="tabular-nums">
                      {u.isSuperAdmin ? "全部" : u.permissions.length}
                    </AdminTd>
                    <AdminTd className="whitespace-nowrap text-slate-500">
                      {new Date(u.createdAt).toLocaleDateString("zh-CN")}
                    </AdminTd>
                    <AdminTd className="space-x-2 text-right">
                      <button
                        type="button"
                        onClick={() => openEdit(u)}
                        className="text-sm font-medium text-blue-600 hover:text-blue-800"
                      >
                        编辑
                      </button>
                      {!u.isSuperAdmin && (
                        <button
                          type="button"
                          onClick={() => handleDelete(u)}
                          className="text-sm font-medium text-red-600 hover:text-red-800"
                        >
                          删除
                        </button>
                      )}
                    </AdminTd>
                  </tr>
                ))
              )}
            </AdminTableBody>
          </AdminTable>

          <AdminPagination
            page={page}
            totalPages={totalPages}
            total={total}
            onPageChange={setPage}
          />
        </>
      )}

      {/* Create/Edit Dialog */}
      {showDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/50" onClick={() => setShowDialog(false)} />
          <div className="relative bg-white rounded-xl shadow-xl w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6 space-y-4">
              <h2 className="text-lg font-bold text-gray-900">
                {editingUser ? "编辑用户" : "创建用户"}
              </h2>

              {formError && (
                <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm">{formError}</div>
              )}

              {/* Username */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">用户名</label>
                <input
                  type="text"
                  value={form.username}
                  onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder="输入用户名"
                  disabled={editingUser?.isSuperAdmin}
                />
              </div>

              {/* Password */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  密码{editingUser ? "（留空不修改）" : ""}
                </label>
                <input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  placeholder={editingUser ? "留空不修改" : "输入密码（至少6位）"}
                />
              </div>

              {/* Role */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">角色</label>
                <select
                  value={form.role}
                  onChange={(e) => setForm((f) => ({ ...f, role: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  disabled={editingUser?.isSuperAdmin}
                >
                  <option value="admin">管理员</option>
                  <option value="editor">编辑</option>
                </select>
              </div>

              {/* Permissions */}
              {!editingUser?.isSuperAdmin && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="block text-sm font-medium text-gray-700">权限</label>
                    <button
                      type="button"
                      onClick={toggleAllPermissions}
                      className="text-xs text-blue-600 hover:text-blue-800"
                    >
                      {ALL_PERMISSIONS.every((p) => form.permissions.includes(p.key))
                        ? "取消全选"
                        : "全选"}
                    </button>
                  </div>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                    {ALL_PERMISSIONS.map((perm) => (
                      <label
                        key={perm.key}
                        className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer hover:bg-gray-50 px-2 py-1.5 rounded"
                      >
                        <input
                          type="checkbox"
                          checked={form.permissions.includes(perm.key)}
                          onChange={() => togglePermission(perm.key)}
                          className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        />
                        {perm.label}
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {editingUser?.isSuperAdmin && (
                <div className="p-3 bg-amber-50 text-amber-700 rounded-lg text-sm">
                  超级管理员拥有全部权限，不可修改权限设置。
                </div>
              )}

              {/* Actions */}
              <div className="flex justify-end gap-3 pt-2">
                <button
                  onClick={() => setShowDialog(false)}
                  className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50"
                >
                  取消
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {saving ? "保存中..." : "保存"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
