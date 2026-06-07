function handler(event) {
  var request = event.request;
  var uri = request.uri;

  if (uri.endsWith('/')) {
    request.uri = uri + 'index.html';
  } else if (!uri.includes('.')) {
    // SEO pages are stored as .html files (e.g. /contracts/naics/541330.html)
    if (uri.startsWith('/contracts/')) {
      request.uri = uri + '.html';
    } else {
      request.uri = uri + '/index.html';
    }
  }

  return request;
}
