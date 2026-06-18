"use client";

import { useRef, useState } from "react";
import { showAlert } from "@/app/components/Dialog";

// ── CSV helpers ───────────────────────────────────────────────────────────────

function escapeCell(v: unknown): string {
  const s = v == null ? "" : String(v);
  if (s.includes(",") || s.includes('"') || s.includes("\n")) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

function toCSV(headers: string[], rows: Record<string, unknown>[]): string {
  const lines = [headers.join(",")];
  for (const row of rows) {
    lines.push(headers.map((h) => escapeCell(row[h])).join(","));
  }
  return lines.join("\n");
}

function downloadCSV(filename: string, csv: string) {
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function parseCSV(text: string): { headers: string[]; rows: Record<string, string>[] } {
  const lines = text.trim().split(/\r?\n/);
  if (lines.length < 1) return { headers: [], rows: [] };
  const headers = lines[0].split(",").map((h) => h.trim().replace(/^"|"$/g, ""));
  const rows: Record<string, string>[] = [];
  for (let i = 1; i < lines.length; i++) {
    const cells = splitCSVLine(lines[i]);
    const row: Record<string, string> = {};
    headers.forEach((h, idx) => { row[h] = (cells[idx] ?? "").trim(); });
    rows.push(row);
  }
  return { headers, rows };
}

function splitCSVLine(line: string): string[] {
  const cells: string[] = [];
  let cur = "", inQ = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      if (inQ && line[i + 1] === '"') { cur += '"'; i++; }
      else { inQ = !inQ; }
    } else if (c === "," && !inQ) {
      cells.push(cur); cur = "";
    } else {
      cur += c;
    }
  }
  cells.push(cur);
  return cells;
}

// ── Export button ─────────────────────────────────────────────────────────────

interface ExportProps<T extends Record<string, unknown>> {
  filename: string;
  headers: string[];
  /** Called to fetch all data; should loop pages and return all rows. */
  fetchAll: () => Promise<T[]>;
}

export function ExportButton<T extends Record<string, unknown>>({ filename, headers, fetchAll }: ExportProps<T>) {
  const [busy, setBusy] = useState(false);

  async function run() {
    setBusy(true);
    try {
      const rows = await fetchAll();
      const csv = toCSV(headers, rows as Record<string, unknown>[]);
      downloadCSV(filename, csv);
    } catch (e: unknown) {
      await showAlert(e instanceof Error ? e.message : "Export failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <button className="btn btn-ghost" onClick={run} disabled={busy} style={{ fontSize: "0.8125rem" }}>
      {busy ? "Exporting…" : "↓ Export CSV"}
    </button>
  );
}

// ── Import modal ──────────────────────────────────────────────────────────────

interface ImportResult { ok: number; errors: { row: number; message: string }[] }

interface ImportProps {
  title: string;
  /** Expected CSV column names — first row must contain exactly these. */
  expectedHeaders: string[];
  /** Template values for the blank template row (shown as hints). */
  templateRow: Record<string, string>;
  /** Called once per CSV row; should POST and return on success or throw on error. */
  importRow?: (row: Record<string, string>) => Promise<void>;
  /**
   * Alternative to importRow: called once with ALL parsed rows.
   * Return ok count + any row-level errors.
   */
  importAll?: (rows: Record<string, string>[]) => Promise<{ ok: number; errors: { row: number; message: string }[] }>;
  // TODO(csv-import-service): csvUpload prop (multipart POST) will be
  // re-added once the async import-service handles large CSV jobs.
  onDone: () => void;
}

export function ImportButton(props: ImportProps) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button className="btn btn-ghost" onClick={() => setOpen(true)} style={{ fontSize: "0.8125rem" }}>
        ↑ Import CSV
      </button>
      {open && <ImportModal {...props} onClose={() => setOpen(false)} />}
    </>
  );
}

function ImportModal({ title, expectedHeaders, templateRow, importRow, importAll, onDone, onClose }: ImportProps & { onClose: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [parsed, setParsed] = useState<{ headers: string[]; rows: Record<string, string>[] } | null>(null);
  const [parseErr, setParseErr] = useState("");
  const [importing, setImporting] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragging, setDragging] = useState(false);

  function downloadTemplate() {
    const csv = toCSV(expectedHeaders, [templateRow]);
    downloadCSV(`${title.toLowerCase().replace(/\s+/g, "-")}-template.csv`, csv);
  }

  function handleFile(file: File) {
    setParseErr("");
    setParsed(null);
    setResult(null);
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      const p = parseCSV(text);
      const missing = expectedHeaders.filter((h) => !p.headers.includes(h));
      if (missing.length) {
        setParseErr(`Missing required columns: ${missing.join(", ")}`);
        return;
      }
      setParsed(p);
    };
    reader.readAsText(file);
  }

  function onDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFile(file);
  }

  async function runImport() {
    if (!parsed) return;
    setImporting(true);
    setProgress(0);

    let result: ImportResult;

    if (importAll) {
      try {
        result = await importAll(parsed.rows);
      } catch (e: unknown) {
        result = { ok: 0, errors: [{ row: 0, message: e instanceof Error ? e.message : "Import failed" }] };
      }
      setProgress(parsed.rows.length);
    } else {
      // Row-by-row fallback
      const errors: ImportResult["errors"] = [];
      for (let i = 0; i < parsed.rows.length; i++) {
        try {
          await importRow!(parsed.rows[i]);
        } catch (e: unknown) {
          errors.push({ row: i + 2, message: e instanceof Error ? e.message : "Unknown error" });
        }
        setProgress(i + 1);
      }
      result = { ok: parsed.rows.length - errors.length, errors };
    }

    setImporting(false);
    setResult(result);
    onDone();
  }

  const total = parsed?.rows.length ?? 0;
  const pct = total > 0 ? Math.round((progress / total) * 100) : 0;

  return (
    <div className="modal-overlay" onClick={!importing ? onClose : undefined}>
      <div className="modal" style={{ maxWidth: 560 }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">Import {title}</h2>
          <p className="modal-subtitle">
            Upload a CSV file — or{" "}
            <button
              onClick={downloadTemplate}
              style={{ background: "none", border: "none", color: "var(--accent)", cursor: "pointer", padding: 0, fontSize: "inherit" }}
            >
              download the template
            </button>
          </p>
        </div>
        <div className="modal-body">
          {!result ? (
            <div style={{ display: "flex", flexDirection: "column", gap: "1rem" }}>
              {/* Drop zone */}
              {!parsed && (
                <div
                  onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
                  onDragLeave={() => setDragging(false)}
                  onDrop={onDrop}
                  onClick={() => fileRef.current?.click()}
                  style={{
                    border: `2px dashed ${dragging ? "var(--accent)" : "var(--border)"}`,
                    borderRadius: 8,
                    padding: "2rem 1rem",
                    textAlign: "center",
                    cursor: "pointer",
                    background: dragging ? "var(--accent-bg)" : "var(--surface2)",
                    transition: "border-color 0.15s, background 0.15s",
                  }}
                >
                  <div style={{ fontSize: "1.5rem", marginBottom: "0.375rem" }}>📂</div>
                  <p style={{ margin: 0, fontSize: "0.875rem", color: "var(--text-2)" }}>
                    Drop CSV here or <span style={{ color: "var(--accent)" }}>click to browse</span>
                  </p>
                  <p style={{ margin: "0.25rem 0 0", fontSize: "0.75rem", color: "var(--muted)" }}>
                    Required columns: {expectedHeaders.join(", ")}
                  </p>
                  <input ref={fileRef} type="file" accept=".csv,text/csv" style={{ display: "none" }}
                    onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }} />
                </div>
              )}

              {parseErr && <p style={{ color: "var(--danger)", fontSize: "0.8125rem", margin: 0 }}>{parseErr}</p>}

              {/* Preview */}
              {parsed && !importing && (
                <>
                  <div style={{ fontSize: "0.8125rem", color: "var(--text-2)", display: "flex", gap: "0.75rem", alignItems: "center" }}>
                    <span style={{ color: "var(--text)", fontWeight: 600 }}>{parsed.rows.length} rows</span> detected
                    <button
                      className="btn btn-ghost"
                      style={{ fontSize: "0.75rem", padding: "0.2rem 0.5rem", marginLeft: "auto" }}
                      onClick={() => { setParsed(null); if (fileRef.current) fileRef.current.value = ""; }}
                    >
                      Change file
                    </button>
                  </div>
                  <div style={{ overflowX: "auto", borderRadius: 6, border: "1px solid var(--border)" }}>
                    <table style={{ fontSize: "0.75rem" }}>
                      <thead>
                        <tr>{parsed.headers.map((h) => <th key={h}>{h}</th>)}</tr>
                      </thead>
                      <tbody>
                        {(() => {
                          // Show first 2 rows + up to 3 rows that have the most filled columns
                          // so optional columns (like parent_name) are visible in the preview.
                          const score = (r: Record<string, string>) =>
                            parsed.headers.filter((h) => r[h]).length;
                          const head = parsed.rows.slice(0, 2);
                          const rest = parsed.rows.slice(2);
                          const rich = [...rest]
                            .sort((a, b) => score(b) - score(a))
                            .slice(0, 3);
                          // Deduplicate while preserving order
                          const seen = new Set(head.map((_, i) => i));
                          const preview = [...head];
                          for (const r of rich) {
                            const idx = parsed.rows.indexOf(r);
                            if (!seen.has(idx)) { seen.add(idx); preview.push(r); }
                          }
                          return preview.map((row, i) => (
                            <tr key={i}>
                              {parsed.headers.map((h) => (
                                <td key={h} style={{ color: row[h] ? "var(--text-2)" : "var(--muted)", fontStyle: row[h] ? "normal" : "italic" }}>
                                  {row[h] || "—"}
                                </td>
                              ))}
                            </tr>
                          ));
                        })()}
                        {parsed.rows.length > 5 && (
                          <tr><td colSpan={parsed.headers.length} style={{ color: "var(--muted)", fontStyle: "italic" }}>
                            … and {parsed.rows.length - 5} more rows
                          </td></tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </>
              )}

              {/* Progress bar */}
              {importing && (
                <div>
                  <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "0.375rem", fontSize: "0.8125rem", color: "var(--text-2)" }}>
                    <span>Importing…</span>
                    <span>{progress} / {total}</span>
                  </div>
                  <div style={{ background: "var(--surface2)", borderRadius: 4, height: 8, overflow: "hidden" }}>
                    <div style={{ background: "var(--accent)", height: "100%", width: `${pct}%`, transition: "width 0.2s" }} />
                  </div>
                </div>
              )}

              <div style={{ display: "flex", gap: "0.625rem", justifyContent: "flex-end" }}>
                <button className="btn btn-ghost" onClick={onClose} disabled={importing}>Cancel</button>
                <button
                  className="btn btn-primary"
                  onClick={runImport}
                  disabled={!parsed || importing}
                >
                  {importing ? `Importing ${progress}/${total}…` : `Import ${total} row${total === 1 ? "" : "s"} →`}
                </button>
              </div>
            </div>
          ) : (
            /* Result summary */
            <div style={{ display: "flex", flexDirection: "column", gap: "0.875rem" }}>
              <div style={{ display: "flex", gap: "1rem" }}>
                <div className="card" style={{ padding: "0.75rem 1rem", flex: 1, textAlign: "center" }}>
                  <div style={{ fontSize: "1.5rem", fontWeight: 700, color: "var(--success)" }}>{result.ok}</div>
                  <div style={{ fontSize: "0.75rem", color: "var(--muted)" }}>imported</div>
                </div>
                <div className="card" style={{ padding: "0.75rem 1rem", flex: 1, textAlign: "center" }}>
                  <div style={{ fontSize: "1.5rem", fontWeight: 700, color: result.errors.length ? "var(--danger)" : "var(--muted)" }}>{result.errors.length}</div>
                  <div style={{ fontSize: "0.75rem", color: "var(--muted)" }}>errors</div>
                </div>
              </div>
              {result.errors.length > 0 && (
                <div style={{ maxHeight: 180, overflowY: "auto", border: "1px solid var(--border)", borderRadius: 6 }}>
                  <table style={{ fontSize: "0.75rem" }}>
                    <thead><tr><th>Row</th><th>Error</th></tr></thead>
                    <tbody>
                      {result.errors.map((e, i) => (
                        <tr key={i}>
                          <td style={{ color: "var(--muted)" }}>#{e.row}</td>
                          <td style={{ color: "var(--danger)" }}>{e.message}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <div style={{ display: "flex", justifyContent: "flex-end" }}>
                <button className="btn btn-primary" onClick={onClose}>Done</button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
