document.addEventListener('DOMContentLoaded', function () {
  var params = new URLSearchParams(window.location.search);
  var source = params.get('utm_source');
  var medium = params.get('utm_medium');
  var campaign = params.get('utm_campaign');

  if (!source && !medium && !campaign) return;

  var qs = '';
  if (source) qs += '&utm_source=' + encodeURIComponent(source);
  if (medium) qs += '&utm_medium=' + encodeURIComponent(medium);
  if (campaign) qs += '&utm_campaign=' + encodeURIComponent(campaign);
  qs = qs.substring(1);

  document.querySelectorAll('a[href*="app.govtrove.com"]').forEach(function (a) {
    var href = a.getAttribute('href');
    a.setAttribute('href', href + (href.indexOf('?') === -1 ? '?' : '&') + qs);
  });

  try {
    var payload = JSON.stringify({
      utm_source: source,
      utm_medium: medium,
      utm_campaign: campaign,
      landing_page: window.location.pathname,
      origin: 'landing',
      referrer: document.referrer || null,
    });
    var xhr = new XMLHttpRequest();
    xhr.open('POST', 'https://api.govtrove.com/api/utm', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.send(payload);
  } catch (e) {}
});
