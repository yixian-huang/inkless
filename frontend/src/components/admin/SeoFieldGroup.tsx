interface SeoFieldGroupProps {
  seoTitle: string;
  onSeoTitleChange: (value: string) => void;
  metaDescription: string;
  onMetaDescriptionChange: (value: string) => void;
  ogImage?: string;
  onOgImageChange?: (value: string) => void;
  keywords?: string;
  onKeywordsChange?: (value: string) => void;
  label?: string;
}

export default function SeoFieldGroup({
  seoTitle,
  onSeoTitleChange,
  metaDescription,
  onMetaDescriptionChange,
  ogImage,
  onOgImageChange,
  keywords,
  onKeywordsChange,
  label = "SEO",
}: SeoFieldGroupProps) {
  return (
    <div className="space-y-3">
      <h4 className="text-sm font-medium text-gray-700">{label}</h4>
      <div>
        <label className="block text-xs text-gray-500 mb-1">SEO Title</label>
        <input
          type="text"
          value={seoTitle}
          onChange={(e) => onSeoTitleChange(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm"
          placeholder="Override page title for search engines"
          maxLength={70}
        />
        <span className="text-xs text-gray-400">{seoTitle.length}/70</span>
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">Meta Description</label>
        <textarea
          value={metaDescription}
          onChange={(e) => onMetaDescriptionChange(e.target.value)}
          className="w-full border rounded px-3 py-1.5 text-sm"
          rows={2}
          placeholder="Description for search engine results"
          maxLength={160}
        />
        <span className="text-xs text-gray-400">{metaDescription.length}/160</span>
      </div>
      {onKeywordsChange && (
        <div>
          <label className="block text-xs text-gray-500 mb-1">Keywords</label>
          <input
            type="text"
            value={keywords ?? ""}
            onChange={(e) => onKeywordsChange(e.target.value)}
            className="w-full border rounded px-3 py-1.5 text-sm"
            placeholder="Comma-separated keywords"
          />
        </div>
      )}
      {onOgImageChange && (
        <div>
          <label className="block text-xs text-gray-500 mb-1">OG Image URL</label>
          <input
            type="text"
            value={ogImage ?? ""}
            onChange={(e) => onOgImageChange(e.target.value)}
            className="w-full border rounded px-3 py-1.5 text-sm"
            placeholder="Image for social sharing preview"
          />
        </div>
      )}
    </div>
  );
}
