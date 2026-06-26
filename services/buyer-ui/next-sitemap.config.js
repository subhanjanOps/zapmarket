/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: process.env.NEXT_PUBLIC_SITE_URL ?? "https://zapmarket.in",
  generateRobotsTxt: true,
  exclude: ["/api/*", "/account/*", "/checkout", "/cart"],
  robotsTxtOptions: {
    policies: [
      { userAgent: "*", allow: "/" },
      { userAgent: "*", disallow: ["/api/", "/account/", "/checkout", "/cart"] },
    ],
  },
};
