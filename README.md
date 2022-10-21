# sico - A Sitemap comparison tool

A Sitemap Comparison that helps you to not fuck up your website migration.

## Usecase

### Website migration

Imagine you want to migrate a website.
Lets say, your personal website https://andygrunwald.com/ (incl. a blog) from [Hugo](https://gohugo.io/) to [Astro](https://astro.build/).
You have a few blog posts that rank pretty well in Google - You want to keep all previous URLs.

This tool checks if all URLs from the old website's sitemap are present in the new one.
It is not doing a 1:1 check!
The new site can contain more links in the sitemap.

## Usage

```
Usage of ./sico:
  -exclude value
        Regex to match against URLs in {source} sitemap that don't need to be in {new} sitemap. It can be defined multiple times.
  -new string
        New Sitemap URL - Sitemap entries you want to check for presence (default "https://example-new.com/sitemap.xml")
  -newBaseURL new
        Base URL that will be used if new contains a SitemapIndex to replace the SitemapIndex entries
  -source string
        Source Sitemap URL - Sitemap you want to check against (default "https://example.com/sitemap.xml")
```

### An example

The call ...

```sh
./sico -source "https://andygrunwald.com/sitemap.xml" \
       -new "https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/sitemap-index.xml" \
       -newBaseURL "https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/" \
       -exclude "andygrunwald\\.com/tags/"
```

... means:

1. We read the `-source` sitemap from `https://andygrunwald.com/sitemap.xml` and collect the URLs:
    ```xml
    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
        <url>
            <loc>https://andygrunwald.com/</loc>
            <lastmod>2021-09-21T20:30:00+02:00</lastmod>
        </url>
        <url>
        [...]
    </<urlset>
    ```
2. We read the `-new` sitemap from `https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/sitemap-index.xml` and collect the URLs:
    ```xml
    <sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
        <sitemap>
            <loc>https://andygrunwald.com/sitemap-0.xml</loc>
        </sitemap>
    </sitemapindex>
    
    1. If this URL is a SitemapIndex ([one sitemap split into multiple sitemaps](https://developers.google.com/search/docs/crawling-indexing/sitemaps/large-sitemaps))
    2. AND `-newBaseURL` is set, replace the Base URL (Scheme + Host) of the Sub-Sitemap with `-newBaseURL`
    3. Means `https://andygrunwald.com/sitemap-0.xml` will be changed to `https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/sitemap-0.xml`
3. Loop through all URLs from `-source` (`https://andygrunwald.com/sitemap.xml`), check if the URL matches a defined `-exclude` and needs to be skipped; if not check if it is part of the `-new` sitemap. If yes, all good; if not, raise this as output (see below)
4. Result:
    ```
    Result
    =============
    Source Sitemap: https://andygrunwald.com/sitemap.xml
    URLs checked (from source sitemap): 60
    New Sitemap: https://deploy-preview-7--spiffy-shortbread-df2800.netlify.app/sitemap-index.xml
    Excludes configured: 1

    URLs skipped because they matched an exclude: 24
    URLs missing from source sitemap in new sitemap: 14

    Missing URLs in the new sitemap:
    https://andygrunwald.com/categories/
    https://andygrunwald.com/categories/hardware/
    [...]
    ```

## Production ready?

No, not really.
But it does the job.

This tool was created "on the get-go".
It has no focus on reliability, proper error handling, or things that fit into the *production ready* category.
However, this is (partially) not needed.

This tool only reads data from the web and compares it.
No write functionality or anything else.
Hence, no damage.

Means: You can see it as a (kind of) production ready.