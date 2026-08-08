function handler(event) {
  var request = event.request;
  var uri = request.uri;

  if (uri.endsWith('/')) {
    request.uri = uri + 'index.html';
    return request;
  }

  if (uri.includes('.')) {
    return request;
  }

  // SEO pages are flat objects (/contracts/naics/541330.html), not directories.
  // Serving the extensionless form as 200 made every page reachable at two URLs,
  // so Google filed ~1,000 of them as duplicates and burned crawl budget on them.
  // Redirect instead of rewrite so the .html form is the only indexable URL.
  if (uri.startsWith('/contracts/')) {
    return {
      statusCode: 301,
      statusDescription: 'Moved Permanently',
      headers: { location: { value: uri + '.html' } },
    };
  }

  request.uri = uri + '/index.html';
  return request;
}
