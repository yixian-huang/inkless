import { useState, useEffect, useCallback, Fragment } from "react";
import {
  triggerQAIndex,
  getQALogs,
  submitQAFeedback,
  type QALog,
  type QALogsResponse,
} from "@/api/qa";

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
  return <span className="text-gray-400 text-xs">—</span>;
}

export default function AdminQAPage() {
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
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">知识问答</h2>
        <div className="flex gap-3">
          <button
            onClick={fetchData}
            disabled={loading}
            className="px-4 py-2 bg-gray-100 text-gray-700 text-sm font-medium rounded-md hover:bg-gray-200 disabled:opacity-50"
          >
            {loading ? "加载中..." : "刷新"}
          </button>
          <button
            onClick={handleReindex}
            disabled={indexing}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
          >
            {indexing ? (
              <>
                <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                索引中...
              </>
            ) : "重新索引内容"}
          </button>
        </div>
      </div>

      {indexSuccess && (
        <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-md text-green-700 text-sm">
          {indexSuccess}
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md text-red-700 text-sm">
          {error}
        </div>
      )}

      {loading && !data ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : data ? (
        <>
          <div className="bg-white shadow rounded-lg overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">问题</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">回答</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">评价</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">语言</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">IP</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">时间</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {data.items.map((item: QALog) => {
                  const isExpanded = expandedId === item.id;
                  return (
                    <Fragment key={item.id}>
                      <tr
                        className="hover:bg-gray-50 cursor-pointer"
                        onClick={() => handleToggleExpand(item.id)}
                      >
                        <td className="px-4 py-4 text-sm text-gray-900 max-w-xs">
                          {truncate(item.question)}
                        </td>
                        <td className="px-4 py-4 text-sm text-gray-700 max-w-xs">
                          {truncate(item.answer)}
                        </td>
                        <td className="px-4 py-4 text-sm">
                          <RatingBadge rating={item.rating} />
                        </td>
                        <td className="px-4 py-4 text-sm text-gray-500">
                          {item.locale || "—"}
                        </td>
                        <td className="px-4 py-4 text-sm text-gray-500 whitespace-nowrap">
                          {item.ip_address || "—"}
                        </td>
                        <td className="px-4 py-4 text-sm text-gray-500 whitespace-nowrap">
                          {new Date(item.created_at).toLocaleString("zh-CN")}
                        </td>
                        <td className="px-4 py-4 text-sm" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center gap-2">
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
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr>
                          <td colSpan={7} className="px-4 py-4 bg-gray-50">
                            <div className="space-y-3 text-sm">
                              <div>
                                <span className="font-medium text-gray-700">完整问题：</span>
                                <p className="mt-1 text-gray-600 whitespace-pre-wrap">{item.question}</p>
                              </div>
                              <div>
                                <span className="font-medium text-gray-700">完整回答：</span>
                                <p className="mt-1 text-gray-600 whitespace-pre-wrap">{item.answer}</p>
                              </div>
                              {item.sources && (
                                <div>
                                  <span className="font-medium text-gray-700">来源：</span>
                                  <p className="mt-1 text-gray-600 text-xs font-mono bg-white border border-gray-200 rounded p-2 whitespace-pre-wrap">
                                    {JSON.stringify(item.sources, null, 2)}
                                  </p>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
                {data.items.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-6 py-8 text-center text-sm text-gray-500">
                      暂无问答记录
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-gray-500">
                共 {data.total} 条，第 {page}/{totalPages} 页
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  上一页
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </>
      ) : null}
    </div>
  );
}
