import { useEffect, useRef } from "react";
import type { ThemeTokens } from "@/theme/tokens";
import {
  DEFAULT_MONO_PRESET_ID,
  DEFAULT_SANS_PRESET_ID,
  DEFAULT_SERIF_PRESET_ID,
  getFontPreset,
  loadCustomFonts,
  presetsForRole,
  resolveArticleTypography,
  typographyToCssVars,
} from "@/theme/typography";
import { uploadMedia } from "@/api/media";
import {
  AdminButton,
  AdminCard,
  AdminField,
  AdminInput,
  AdminSelect,
  AdminTextarea,
} from "@/components/admin/ui";

interface FontPresetSectionProps {
  tokens: ThemeTokens;
  onChange: (tokens: ThemeTokens) => void;
}

function applyPreset(
  tokens: ThemeTokens,
  role: "sans" | "heading" | "mono",
  presetId: string,
): ThemeTokens {
  const preset = getFontPreset(presetId);
  if (!preset) return tokens;

  const fontSources = { ...(tokens.fontSources ?? {}) };
  if (role === "sans") fontSources.sansPresetId = presetId;
  if (role === "heading") fontSources.headingPresetId = presetId;
  if (role === "mono") fontSources.monoPresetId = presetId;

  const fonts = { ...tokens.fonts };
  if (role === "sans") fonts.sans = preset.stack;
  if (role === "heading") fonts.heading = preset.stack;
  if (role === "mono") fonts.mono = preset.stack;

  return { ...tokens, fonts, fontSources };
}

export default function FontPresetSection({ tokens, onChange }: FontPresetSectionProps) {
  const uploadRef = useRef<HTMLInputElement>(null);
  const sources = tokens.fontSources ?? {};
  const previewConfig = resolveArticleTypography({ tokens });
  const previewStyle = typographyToCssVars(previewConfig);

  useEffect(() => {
    loadCustomFonts(previewConfig.customFonts);
  }, [previewConfig.customFonts]);

  const sansPreset = sources.sansPresetId ?? DEFAULT_SANS_PRESET_ID;
  const headingPreset = sources.headingPresetId ?? DEFAULT_SERIF_PRESET_ID;
  const monoPreset = sources.monoPresetId ?? DEFAULT_MONO_PRESET_ID;

  const handleFontStackChange = (key: keyof ThemeTokens["fonts"], value: string) => {
    onChange({ ...tokens, fonts: { ...tokens.fonts, [key]: value } });
  };

  const handleUpload = async (role: "heading" | "sans", file: File) => {
    const item = await uploadMedia(file);
    const family = window.prompt("请输入字体族名称（CSS font-family，如 My Serif）", "Custom Serif");
    if (!family?.trim()) return;

    const fontSources = { ...(tokens.fontSources ?? {}) };
    const ref = { url: item.url, family: family.trim(), weight: 400, style: "normal" as const };
    if (role === "heading") {
      fontSources.headingUpload = ref;
      onChange({
        ...tokens,
        fontSources,
        fonts: {
          ...tokens.fonts,
          heading: `"${family.trim()}", ${tokens.fonts.heading}`,
        },
      });
    } else {
      fontSources.sansUpload = ref;
      onChange({
        ...tokens,
        fontSources,
        fonts: {
          ...tokens.fonts,
          sans: `"${family.trim()}", ${tokens.fonts.sans}`,
        },
      });
    }
  };

  const handleTypographyChange = (patch: { bodySize?: string; bodyLineHeight?: number }) => {
    onChange({
      ...tokens,
      typography: {
        ...tokens.typography,
        article: {
          ...tokens.typography?.article,
          ...patch,
        },
      },
    });
  };

  const bodySize = tokens.typography?.article?.bodySize ?? "1.0625rem";
  const bodyLineHeight = tokens.typography?.article?.bodyLineHeight ?? 1.8;

  return (
    <AdminCard
      className="mb-6"
      title="字体与文章排版"
      description="字体栈、字号与行高在此统一配置；正文使用衬线或无衬线见「主题设置 → 文章 → 正文默认字体」。"
    >
      <div className="mb-6 grid grid-cols-1 gap-6 lg:grid-cols-3">
        <PresetSelect
          label="UI / 无衬线 (Sans)"
          labelZh="界面与元信息"
          value={sansPreset}
          options={presetsForRole("sans")}
          onChange={(id) => onChange(applyPreset(tokens, "sans", id))}
        />
        <PresetSelect
          label="正文衬线 (Heading token)"
          labelZh="标题与衬线正文"
          value={headingPreset}
          options={presetsForRole("serif")}
          onChange={(id) => onChange(applyPreset(tokens, "heading", id))}
        />
        <PresetSelect
          label="代码 (Mono)"
          labelZh="代码块"
          value={monoPreset}
          options={presetsForRole("mono")}
          onChange={(id) => onChange(applyPreset(tokens, "mono", id))}
        />
      </div>

      <div className="mb-6 grid grid-cols-1 gap-4 border-t border-slate-100 pt-4 md:grid-cols-2">
        <AdminField label="正文字号（CSS）">
          <AdminInput
            type="text"
            value={bodySize}
            onChange={(e) => handleTypographyChange({ bodySize: e.target.value })}
            className="font-mono"
            placeholder="1.0625rem"
          />
        </AdminField>
        <AdminField label="正文行高">
          <AdminInput
            type="number"
            step="0.05"
            min="1"
            max="2.5"
            value={bodyLineHeight}
            onChange={(e) => handleTypographyChange({ bodyLineHeight: Number(e.target.value) })}
          />
        </AdminField>
      </div>

      <div className="mb-6 flex flex-wrap gap-2">
        <AdminButton
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => {
            uploadRef.current?.setAttribute("data-role", "heading");
            uploadRef.current?.click();
          }}
        >
          上传衬线字体 (woff2)
        </AdminButton>
        <AdminButton
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => {
            uploadRef.current?.setAttribute("data-role", "sans");
            uploadRef.current?.click();
          }}
        >
          上传无衬线字体 (woff2)
        </AdminButton>
        <input
          ref={uploadRef}
          type="file"
          accept=".woff2,.woff,font/woff2,font/woff"
          className="hidden"
          onChange={async (e) => {
            const file = e.target.files?.[0];
            const role = (e.target.getAttribute("data-role") ?? "heading") as "heading" | "sans";
            e.target.value = "";
            if (!file) return;
            try {
              await handleUpload(role, file);
            } catch {
              window.alert("字体上传失败");
            }
          }}
        />
      </div>

      <details className="text-sm">
        <summary className="mb-2 cursor-pointer font-medium text-slate-600">
          高级：自定义 font-family 栈
        </summary>
        <div className="mt-2 grid grid-cols-1 gap-4 md:grid-cols-2">
          <AdminField label="Sans 栈">
            <AdminTextarea
              value={tokens.fonts.sans}
              onChange={(e) => handleFontStackChange("sans", e.target.value)}
              rows={2}
              className="font-mono text-xs"
            />
          </AdminField>
          <AdminField label="Heading（衬线）栈">
            <AdminTextarea
              value={tokens.fonts.heading}
              onChange={(e) => handleFontStackChange("heading", e.target.value)}
              rows={2}
              className="font-mono text-xs"
            />
          </AdminField>
        </div>
      </details>

      <div
        className="article-typography article-reading bg-surface mt-6 rounded-2xl border border-slate-200 p-4"
        style={previewStyle}
      >
        <p className="article-page-title mb-2 text-xl">文章标题预览 Sample Title</p>
        <div className="tiptap ProseMirror">
          <p>
            中文正文预览：这是一段用于检验字号与行高的示例文字。The quick brown fox jumps over the lazy
            dog.
          </p>
          <blockquote>引用块应使用 muted 色与细左边框。</blockquote>
          <pre>
            <code>const mono = true;</code>
          </pre>
        </div>
      </div>
    </AdminCard>
  );
}

function PresetSelect({
  label,
  labelZh,
  value,
  options,
  onChange,
}: {
  label: string;
  labelZh: string;
  value: string;
  options: ReturnType<typeof presetsForRole>;
  onChange: (id: string) => void;
}) {
  return (
    <AdminField label={labelZh} hint={label}>
      <AdminSelect value={value} onChange={(e) => onChange(e.target.value)} className="w-full">
        {options.map((p) => (
          <option key={p.id} value={p.id}>
            {p.nameZh}
          </option>
        ))}
      </AdminSelect>
    </AdminField>
  );
}
