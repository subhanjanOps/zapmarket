import fs from "fs";
import path from "path";
import matter from "gray-matter";

const blogDir = path.join(process.cwd(), "content/blog");
const glossaryDir = path.join(process.cwd(), "content/glossary");

export interface PostMeta {
  title: string;
  slug: string;
  excerpt: string;
  category: string;
  product_slugs: string[];
  published_at: string;
  featured: boolean;
}

export interface TermMeta {
  term: string;
  slug: string;
  category: string;
  related_terms: string[];
  product_slugs: string[];
}

function readDir(dir: string): string[] {
  try {
    return fs.readdirSync(dir).filter((f) => f.endsWith(".mdx"));
  } catch {
    return [];
  }
}

// --- Posts cache ---

let _postsCache: PostMeta[] | null = null;
let _postMap: Map<string, { meta: PostMeta; content: string }> | null = null;

function buildPostMap(): Map<string, { meta: PostMeta; content: string }> {
  if (_postMap) return _postMap;
  _postMap = new Map();
  for (const f of readDir(blogDir)) {
    const raw = fs.readFileSync(path.join(blogDir, f), "utf8");
    const { data, content } = matter(raw);
    _postMap.set(data.slug, { meta: data as PostMeta, content });
  }
  return _postMap;
}

export function getAllPosts(): PostMeta[] {
  if (_postsCache) return _postsCache;
  _postsCache = Array.from(buildPostMap().values())
    .map((v) => v.meta)
    .sort((a, b) => (a.published_at < b.published_at ? 1 : -1));
  return _postsCache;
}

export function getPost(slug: string): { meta: PostMeta; content: string } | null {
  return buildPostMap().get(slug) ?? null;
}

// --- Terms cache ---

let _termsCache: TermMeta[] | null = null;
let _termMap: Map<string, { meta: TermMeta; content: string }> | null = null;

function buildTermMap(): Map<string, { meta: TermMeta; content: string }> {
  if (_termMap) return _termMap;
  _termMap = new Map();
  for (const f of readDir(glossaryDir)) {
    const raw = fs.readFileSync(path.join(glossaryDir, f), "utf8");
    const { data, content } = matter(raw);
    _termMap.set(data.slug, { meta: data as TermMeta, content });
  }
  return _termMap;
}

export function getAllTerms(): TermMeta[] {
  if (_termsCache) return _termsCache;
  _termsCache = Array.from(buildTermMap().values()).map((v) => v.meta);
  return _termsCache;
}

export function getTerm(slug: string): { meta: TermMeta; content: string } | null {
  return buildTermMap().get(slug) ?? null;
}
