/** @type {import('next-sitemap').IConfig} */
module.exports = {
  siteUrl: process.env.SITE_URL ?? "https://zapmarket.com",
  generateRobotsTxt: true,
  exclude: ["/checkout", "/account/*", "/api/*"],
};
