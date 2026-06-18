"use client";
import { useRef, useState } from "react";
import { Upload, X } from "lucide-react";

interface PendingFile { file: File; preview: string; progress: number; done: boolean; error?: string; }

interface Props {
  onUpload: (file: File) => Promise<void>;
}

export function ImageDropzone({ onUpload }: Props) {
  const [pending, setPending] = useState<PendingFile[]>([]);
  const [dragOver, setDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  async function handleFiles(files: FileList | null) {
    if (!files || files.length === 0) return;
    const arr = Array.from(files);
    const newEntries: PendingFile[] = arr.map((f) => ({
      file: f,
      preview: URL.createObjectURL(f),
      progress: 0,
      done: false,
    }));
    setPending((p) => [...p, ...newEntries]);

    for (let i = 0; i < newEntries.length; i++) {
      const entry = newEntries[i];
      const idx = pending.length + i;
      setPending((p) => p.map((x, j) => j === idx ? { ...x, progress: 30 } : x));
      try {
        await onUpload(entry.file);
        setPending((p) => p.map((x, j) => j === idx ? { ...x, progress: 100, done: true } : x));
      } catch (err) {
        const msg = err instanceof Error ? err.message : "Upload failed";
        setPending((p) => p.map((x, j) => j === idx ? { ...x, progress: 0, error: msg } : x));
      }
    }
  }

  function remove(idx: number) {
    setPending((p) => {
      URL.revokeObjectURL(p[idx].preview);
      return p.filter((_, i) => i !== idx);
    });
  }

  return (
    <div>
      <div
        className={`dropzone${dragOver ? " drag-over" : ""}`}
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => { e.preventDefault(); setDragOver(false); handleFiles(e.dataTransfer.files); }}
      >
        <Upload size={20} style={{ color: "var(--muted)", margin: "0 auto 0.5rem" }} />
        <p style={{ margin: 0, fontSize: "0.8125rem", color: "var(--text-2)" }}>
          Drop images here or <span style={{ color: "var(--accent)" }}>browse</span>
        </p>
        <p style={{ margin: "0.25rem 0 0", fontSize: "0.75rem", color: "var(--muted)" }}>
          PNG, JPG, WEBP — max 5 MB each
        </p>
        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          multiple
          style={{ display: "none" }}
          onChange={(e) => handleFiles(e.target.files)}
        />
      </div>

      {pending.length > 0 && (
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(7rem, 1fr))", gap: "0.75rem", marginTop: "1rem" }}>
          {pending.map((p, i) => (
            <div key={i} style={{ position: "relative", borderRadius: 8, overflow: "hidden", border: "1px solid var(--border)", background: "var(--surface2)" }}>
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={p.preview} alt="" style={{ width: "100%", aspectRatio: "1", objectFit: "cover", display: "block" }} />
              {!p.done && !p.error && (
                <div style={{ position: "absolute", inset: 0, background: "rgba(0,0,0,0.5)", display: "flex", alignItems: "center", justifyContent: "center" }}>
                  <div style={{ width: "60%", height: 3, background: "var(--border)", borderRadius: 2 }}>
                    <div style={{ width: `${p.progress}%`, height: "100%", background: "var(--accent)", borderRadius: 2, transition: "width 0.3s" }} />
                  </div>
                </div>
              )}
              {p.error && (
                <div style={{ position: "absolute", inset: 0, background: "rgba(0,0,0,0.6)", display: "flex", alignItems: "center", justifyContent: "center", padding: "0.25rem" }}>
                  <span style={{ fontSize: "0.625rem", color: "var(--danger)", textAlign: "center" }}>{p.error}</span>
                </div>
              )}
              <button
                onClick={(e) => { e.stopPropagation(); remove(i); }}
                style={{ position: "absolute", top: 4, right: 4, background: "rgba(0,0,0,0.6)", border: "none", borderRadius: 4, cursor: "pointer", padding: "2px", display: "flex" }}
              >
                <X size={12} color="#fff" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
