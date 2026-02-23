---
name: blog-post
description: Write and publish a new blog post for the GovTrove landing site
argument-hint: "<topic or title>"
---

# Blog Post

Write and publish a new blog post for the GovTrove landing site.

## Post-Publish Checklist

After creating the blog post HTML file in `landing/blog/`:

1. **Update sitemap** — Add the new post URL to `landing/sitemap.xml` with today's date as `<lastmod>`, `monthly` changefreq, and `0.7` priority
2. **Update blog index** — Add the post to `landing/blog/index.html` so it appears in the blog listing
