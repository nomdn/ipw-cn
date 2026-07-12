import { defineSitemapEventHandler } from '#imports'


export default defineSitemapEventHandler(async () => {
    const config = useRuntimeConfig();
    console.log('Generating sitemap URLs from docConfig:', config.public.docConfig);

    const urls = config.public.docConfig ? Object.keys(config.public.docConfig).map(path => ({ path })) : [];

    return urls.map(url => ({
        loc: url.path,
        _encoded: true,
    }))
})
