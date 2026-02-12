function handler(event) {
  var request = event.request;
  var uri = request.uri;
  var ua = (request.headers['user-agent'] && request.headers['user-agent'].value) || '';

  var match = uri.match(/^\/opportunity\/(\d+)$/);
  if (!match) return request;

  var bots = /Slackbot|Twitterbot|facebookexternalhit|LinkedInBot|Discordbot|WhatsApp|Googlebot|bingbot/i;
  if (!bots.test(ua)) return request;

  return {
    statusCode: 302,
    statusDescription: 'Found',
    headers: {
      location: { value: 'https://api.govtrove.com/og/opportunities/' + match[1] }
    }
  };
}
