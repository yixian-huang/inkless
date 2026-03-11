import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { getComments, postComment, type Comment } from "@/api/comments";

interface CommentSectionProps {
  contentType: "article" | "page";
  contentId: number;
}

function CommentItem({ comment, onReply }: { comment: Comment; onReply: (id: number) => void }) {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="border-l-2 border-gray-200 pl-4 py-3">
      <div className="flex items-center gap-2 mb-1">
        <span className="font-medium text-sm">{comment.authorName}</span>
        <span className="text-xs text-gray-400">{new Date(comment.createdAt).toLocaleDateString()}</span>
        {comment.pinned && <span className="text-xs bg-yellow-100 text-yellow-700 px-1 rounded">Pinned</span>}
      </div>
      <p className="text-sm text-gray-700 mb-2">{comment.content}</p>
      <button onClick={() => onReply(comment.id)} className="text-xs text-blue-600 hover:underline">Reply</button>
      {comment.children && comment.children.length > 0 && (
        <div className="mt-2">
          <button onClick={() => setCollapsed(!collapsed)} className="text-xs text-gray-500 mb-1">
            {collapsed ? `Show ${comment.children.length} replies` : "Hide replies"}
          </button>
          {!collapsed && (
            <div className="space-y-1">
              {comment.children.map((child) => (
                <CommentItem key={child.id} comment={child} onReply={onReply} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function CommentSection({ contentType, contentId }: CommentSectionProps) {
  const [comments, setComments] = useState<Comment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [replyTo, setReplyTo] = useState<number | null>(null);
  const [form, setForm] = useState({ content: "", authorName: "", authorEmail: "" });
  const [submitting, setSubmitting] = useState(false);
  const { t } = useTranslation();

  const loadComments = useCallback(async () => {
    const resp = await getComments(contentType, contentId, page);
    setComments(resp.comments ?? []);
    setTotal(resp.total);
  }, [contentType, contentId, page]);

  useEffect(() => { loadComments(); }, [loadComments]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.content.trim() || !form.authorName.trim()) return;
    setSubmitting(true);
    try {
      await postComment({
        content: form.content,
        authorName: form.authorName,
        authorEmail: form.authorEmail,
        contentType,
        contentId,
        parentId: replyTo ?? undefined,
      });
      setForm({ content: "", authorName: "", authorEmail: "" });
      setReplyTo(null);
      loadComments();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mt-12 border-t pt-8">
      <h3 className="text-lg font-semibold mb-4">{t("comments.title", "Comments")} ({total})</h3>
      <form onSubmit={handleSubmit} className="mb-8 space-y-3 bg-gray-50 p-4 rounded-lg">
        {replyTo && (
          <div className="text-sm text-blue-600">
            Replying to comment #{replyTo}{" "}
            <button type="button" onClick={() => setReplyTo(null)} className="text-gray-400 ml-1">Cancel</button>
          </div>
        )}
        <div className="flex gap-3">
          <input type="text" placeholder={t("comments.name", "Name *")} value={form.authorName}
            onChange={(e) => setForm({ ...form, authorName: e.target.value })} className="border rounded px-3 py-1.5 text-sm flex-1" required />
          <input type="email" placeholder={t("comments.email", "Email (optional)")} value={form.authorEmail}
            onChange={(e) => setForm({ ...form, authorEmail: e.target.value })} className="border rounded px-3 py-1.5 text-sm flex-1" />
        </div>
        <textarea placeholder={t("comments.write", "Write a comment...")} value={form.content}
          onChange={(e) => setForm({ ...form, content: e.target.value })} className="w-full border rounded px-3 py-2 text-sm" rows={3} required />
        <button type="submit" disabled={submitting} className="bg-blue-600 text-white px-4 py-1.5 rounded text-sm hover:bg-blue-700 disabled:opacity-50">
          {submitting ? t("comments.submitting", "Submitting...") : t("comments.submit", "Submit")}
        </button>
      </form>
      <div className="space-y-2">
        {comments.map((c) => (<CommentItem key={c.id} comment={c} onReply={setReplyTo} />))}
      </div>
      {total > 20 && (
        <div className="flex justify-center gap-2 mt-4">
          {page > 1 && (<button onClick={() => setPage(page - 1)} className="px-3 py-1 bg-gray-100 rounded text-sm">Prev</button>)}
          <span className="px-3 py-1 text-sm text-gray-500">Page {page}</span>
          {total > page * 20 && (<button onClick={() => setPage(page + 1)} className="px-3 py-1 bg-gray-100 rounded text-sm">Next</button>)}
        </div>
      )}
    </div>
  );
}
