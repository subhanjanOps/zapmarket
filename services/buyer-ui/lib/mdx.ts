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

export function getAllPosts(): PostMeta[] {
  const files = fs.readdirSync(blogDir).filter((f) => f.endsWith(".mdx"));
  return files
    .map((f) => {
      const raw = fs.readFileSync(path.join(blogDir, f), "utf8");
      const { data } = matter(raw);
      return data as PostMeta;
    })
    .sort((a, b) => (a.published_at < b.published_at ? 1 : -1));
}

export function getPost(slug: string): { meta: PostMeta; content: string } | null {
  const files = fs.readdirSync(blogDir).filter((f) => f.endsWith(".mdx"));
  for (const f of files) {
    const raw = fs.readFileSync(path.join(blogDir, f), "utf8");
    const { data, content } = matter(raw);
    if (data.slug === slug) return { meta: data as PostMeta, content };
  }
  return null;
}

export function getAllTerms(): TermMeta[] {
  const files = fs.readdirSync(glossaryDir).filter((f) => f.endsWith(".mdx"));
  return files.map((f) => {
    const raw = fs.readFileSync(path.join(glossaryDir, f), "utf8");
    const { data } = matter(raw);
    return data as TermMeta;
  });
}

export function getTerm(slug: string): { meta: TermMeta; content: string } | null {
  const files = fs.readdirSync(glossaryDir).filter((f) => f.endsWith(".mdx"));
  for (const f of files) {
    const raw = fs.readFileSync(path.join(glossaryDir, f), "utf8");
    const { data, content } = matter(raw);
    if (data.slug === slug) return { meta: data as TermMeta, content };
  }
  return null;
}
