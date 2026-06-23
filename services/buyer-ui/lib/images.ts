// Rewrites MinIO image URLs so server-side fetches use the internal Docker
// hostname while browsers receive the public localhost URL.
const INTERNAL = process.env.MINIO_INTERNAL_URL ?? "http://zapmarket-minio:9000";
const PUBLIC   = process.env.NEXT_PUBLIC_MINIO_URL ?? "http://localhost:9000";

export function publicImageUrl(url: string | undefined | null): string {
  if (!url) return "/placeholder-product.png";
  // Replace any internal or localhost:9000 variant with the public URL
  return url
    .replace(/http:\/\/zapmarket-minio:\d+/, PUBLIC)
    .replace(/http:\/\/localhost:9000/, PUBLIC);
}

export function internalImageUrl(url: string | undefined | null): string {
  if (!url) return "";
  return url
    .replace(/http:\/\/localhost:9000/, INTERNAL)
    .replace(/http:\/\/zapmarket-minio:\d+/, INTERNAL);
}
