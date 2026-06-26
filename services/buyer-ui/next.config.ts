import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
    remotePatterns: [
      { protocol: "http", hostname: "localhost", port: "9000" },
      { protocol: "http", hostname: "zapmarket-minio", port: "9000" },
      { protocol: "https", hostname: "**.amazonaws.com" },
    ],
  },
};

export default nextConfig;
