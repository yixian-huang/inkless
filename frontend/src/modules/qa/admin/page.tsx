import { useState, useEffect, useCallback, Fragment } from "react";
import {
  triggerQAIndex,
  getQALogs,
  submitQAFeedback,
  type QALog,
  type QALogsResponse,
} from "../api";
import {
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
} from "@/components/admin/ui";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";

const PAGE_SIZE = 20;

function truncate(text: string, max = 80): string {
  if (!text) return "";
  return text.length > max ? text.slice(0, max) + "..." : text;
}

function RatingBadge({ rating }: { rating: string }) {
  if (rating === "positive") {
    return (
      <span className="inline-flex items-center gap-1 text-green-600" title="正面评价">
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
        </svg>
      </span>
    );
  }
  if (rating === "negative") {
    return (
      <span className="inline-flex items-center gap-1 text-red-500" title="负面评价">
        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
          <path d="M18 9.5a1.5 1.5 0 11-3 0v-6a1.5 1.5 0 013 0v6zM14 9.667v-5.43a2 2 0 00-1.105-1.79l-.05-.025A4 4 0 0011.055 2H5.64a2 2 0 00-1.962 1.608l-1.2 6A2 2 0 004.44 12H8v4a2 2 0 002 2 1 1 0 001-1v-.667a4 4 0 01.8-2.4l1.4-1.866a4 4 0 00.8-2.4z" />
        </svg>
      </span>
    );
  }
  return <span className="text-slate-400 text-xs">—</span>;
}

export default function AdminQAPage() {
  useDocumentTitle("知识问答");
  const [data, setData] = useState<QALogsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [indexing, setIndexing] = useState(false);
  const [indexSuccess, setIndexSuccess] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getQALogs(page, PAGE_SIZE);
      setData(result);
    } catch {
      setError("获取问答日志失败，请稍后重试");
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleReindex = async () => {
    setIndexing(true);
    setIndexSuccess(null);
    setError(null);
    try {
      const result = await triggerQAIndex();
      setIndexSuccess(`索引完成，已索引 ${result.chunksStored} 条内容`);
    } catch {
      setError("内容索引失败，请稍后重试");
    } finally {
      setIndexing(false);
    }
  };

  const handleFeedback = async (id: number, rating: "positive" | "negative") => {
    try {
      await submitQAFeedback(id, rating);
      await fetchData();
    } catch {
      setError("提交反馈失败");
    }
  };

  const handleToggleExpand = (id: number) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  const totalPages = data ? Math.ceil(data.total / PAGE_SIZE) : 0;

  return (
    <div>
      <AdminPageHeader
        title="知识问答"
        description="问答日志、反馈与内容索引（实验性）"
        actions={
          <>
            <AdminButton variant="secondary" size="sm" onClick={fetchData} disabled={loading}>
              {loading ? "加载中…" : "刷新"}
            </AdminButton>
            <AdminButton size="sm" onClick={handleReindex} disabled={indexing}>
              {indexing ? "索引中…" : "重新索引内容"}
            </AdminButton>
          </>
        }
      />

      {indexSuccess && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-md text-green-700 text-sm">
          {indexSuccess}
        </div>
      )}

      {error && <AdminErrorBanner message={error} onDismiss={() => setError(null)} />}

      {loading && !data ? (
        <AdminLoading />
      ) : data ? (
        <>
          <AdminTable>
            <AdminTableHead>
              <tr>
                <AdminTh>问题</AdminTh>
                <AdminTh>回答</AdminTh>
                <AdminTh>评价</AdminTh>
                <AdminTh>语言</AdminTh>
                <AdminTh>IP</AdminTh>
                <AdminTh>时间</AdminTh>
                <AdminTh>操作</AdminTh>
              </tr>
            </AdminTableHead>
            <AdminTableBody>
              {data.items.map((item: QALog) => {
                const isExpanded = expandedId === item.id;
                return (
                  <Fragment key={item.id}>
                    <tr
                      className="hover:bg-slate-50/80 cursor-pointer"
                      onClick={() => handleToggleExpand(item.id)}
                    >
                      <AdminTd className="max-w-xs text-slate-900">
                        {truncate(item.question)}
                      </AdminTd>
                      <AdminTd className="max-w-xs text-slate-700">
                        {truncate(item.answer)}
                      </AdminTd>
                      <AdminTd>
                        <RatingBadge rating={item.rating} />
                      </AdminTd>
                      <AdminTd className="text-slate-500">
                        {item.locale || "—"}
                      </AdminTd>
                      <AdminTd className="whitespace-nowrap text-slate-500">
                        {item.ipAddress || "—"}
                      </AdminTd>
                      <AdminTd className="whitespace-nowrap text-slate-500">
                        {new Date(item.createdAt).toLocaleString("zh-CN")}
                      </AdminTd>
                      <AdminTd>
                        <div
                          className="flex items-center gap-2"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            onClick={() => handleFeedback(item.id, "positive")}
                            className="text-green-600 hover:text-green-800"
                            title="正面评价"
                          >
                            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                              <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
                            </svg>
                          </button>
                          <button
                            onClick={() => handleFeedback(item.id, "negative")}
                            className="text-red-500 hover:text-red-700"
                            title="负面评价"
                          >
                            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                              <path d="M18 9.5a1.5 1.5 0 11-3 0v-6a1.5 1.5 0 013 0v6zM14 9.667v-5.43a2 2 0 00-1.105-1.79l-.05-.025A4 4 0 0011.055 2H5.64a2 2 0 00-1.962 1.608l-1.2 6A2 2 0 004.44 12H8v4a2 2 0 002 2 1 1 0 001-1v-.667a4 4 0 01.8-2.4l1.4-1.866a4 4 0 00.8-2.4z" />
                            </svg>
                          </button>
                        </div>
                      </AdminTd>
                    </tr>
                    {isExpanded && (
                      <tr>
                        <AdminTd colSpan={7} className="bg-slate-50">
                          <div className="space-y-3 text-sm">
                            <div>
                              <span className="font-medium text-slate-700">完整问题：</span>
                              <p className="mt-1 text-slate-600 whitespace-pre-wrap">{item.question}</p>
                            </div>
                            <div>
                              <span className="font-medium text-slate-700">完整回答：</span>
                              <p className="mt-1 text-slate-600 whitespace-pre-wrap">{item.answer}</p>
                            </div>
                            {item.sources && (
                              <div>
                                <span className="font-medium text-slate-700">来源：</span>
                                <p className="mt-1 text-slate-600 text-xs font-mono bg-white border border-slate-200 rounded p-2 whitespace-pre-wrap">
                                  {JSON.stringify(item.sources, null, 2)}
                                </p>
                              </div>
                            )}
                          </div>
                        </AdminTd>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
              {data.items.length === 0 && (
                <tr>
                  <AdminTd colSpan={7} className="py-8 text-center text-slate-500">
                    暂无问答记录
                  </AdminTd>
                </tr>
              )}
            </AdminTableBody>
          </AdminTable>

          <AdminPagination
            page={page}
            totalPages={totalPages}
            total={data.total}
            onPageChange={setPage}
          />
        </>
      ) : null}
    </div>
  );
}
