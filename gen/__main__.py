"""
Generate a massive sample crawlable website.
"""
import bottle
import random
import time
from math import floor
from string import letters, digits

N = int(1e1)
random.seed(0)

start = time.time()
lookup, graph, p = range(N), [None] * N, 30

for i in xrange(len(graph)):
    graph[i] = [i-1, i+1] + random.sample(lookup, p)
    random.shuffle(graph[i])

# fix first and last neighbors
for idx in [0, -1]:
    for i, o in enumerate(graph[idx]):
        graph[idx][i] = o % N

del lookup, p
print "Generation Time:", time.time() - start


LINK_TEMPLATE = '<li>\n\t\t\t\t<a href="/page/{0}">Link {0}</a>\n\t\t\t</li>'
HTML_TEMPLATE = """
<!DOCTYPE html>
<html>
\t<head>
\t\t<title>Page {idx}</title>
\t</head>
\t<body>
\t\t<h1>Page {idx}</h1>
\t\t<ul>
\t\t\t{links}
\t\t</ul>
\t</body>
</html>
""".strip()  # \t\t<pre>{{}}</pre>

n = len(str(N)) - 1
pad = lambda x: str(x).rjust(n, '0')

@bottle.route('/')
@bottle.route('/page/<i:int>')
def hello(i=0):
    if not 0 <= i < len(graph):
        bottle.redirect('/')
    start = time.time()
    random.seed(i)
    links = [LINK_TEMPLATE.format(pad(x)) for x in graph[i]]
    data = dict(idx=pad(i), links='\n\t\t\t'.join(links))
    res = HTML_TEMPLATE.format(**data)

    # # Force payload size
    # size = 5002 - len(res)
    # garbage = list(''.join(random.choice(letters + digits) for _ in range(size)))
    # for i in xrange(0, len(garbage), 50):
    #     garbage[i] = '\n'
    # res = res.format(''.join(garbage))

    # # Force duration
    # duration = time.time() - start
    # time.sleep(0.05 - duration)
    # print "creating", duration, time.time() - start
    return res


bottle.run(host='localhost', port=8080, debug=True)
