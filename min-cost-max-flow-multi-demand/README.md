## mcmfmd

### what this solves?

- you have `n` sources and `m` sinks
- each edge from a source to a sink incurs a cost
- you want to send as much flow from the sources to the sinks whilst minimising cost

### example

```python
from flow import *

A = Node("A")
B = Node("B")
C = Node("C")
D = Node("D")
E = Node("E")

sources: Nodes = Nodes({
        A: Capacity(1),
        B: Capacity(1),
        C: Capacity(4),
        })
sinks: Nodes = Nodes({
        D: Capacity(3),
        E: Capacity(3),
        })
costs: Edges = Edges({
        Edge((A, D)): Cost(10),
        Edge((A, E)): Cost(15),
        Edge((B, D)): Cost(3),
        Edge((B, E)): Cost(10),
        Edge((C, D)): Cost(5),
        Edge((C, E)): Cost(10),
        })

solution, avg_cost = match_flows(sources, sinks, costs)
```

![example](assets/example.png)
