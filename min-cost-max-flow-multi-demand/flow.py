from sys import maxsize
from typing import NewType

import networkx as nx
import matplotlib.pyplot as plt

# Graph primitives
Node = NewType("Node", str)
Edge = NewType("Edge", tuple[Node, Node])
Capacity = NewType("Capacity", int) # Cannot be float
Cost = NewType("Cost", int) # Cannot be float

# Graph composites
Nodes = NewType("Nodes", dict[Node, Capacity])
Edges = NewType("Edges", dict[Edge, Cost])

# Solution
AmountMoved = NewType("AmountMoved", int)
AvgCost = NewType("AvgCost", float)
Solution = NewType("Solution", dict[Edge, AmountMoved])

def total_moved(s: Solution) -> AmountMoved:
    return AmountMoved(sum(s.values()))

def match_flows(sources: Nodes, sinks: Nodes, costs: Edges, 
                print_graph=True, filename="graph") -> tuple[Solution, AvgCost]:
    # Create an nx graph, add nodes and edges
    G = nx.DiGraph()
    for node, demand in sources.items():
        G.add_node(node, demand=-demand, layer=0)
    for node, demand in sinks.items():
        G.add_node(node, demand=demand, layer=1)
    for (src, sink), cost in costs.items():
        G.add_edge(src, sink, weight=cost)

    # Add an infinite source or infinite sink for 
    # cases where the capacity does not equal the 
    # demand 
    #
    # demand < 0 --> Infinite source
    # demand > 0 --> Infinite sink
    # demand = 0 --> Isolated node (do nothing)
    demand = sum(sources.values()) - sum(sinks.values())
    
    reservoir = Node("RESERVOIR")
    G.add_node(reservoir, demand=demand)
    
    if demand < 0:
        for sink in sinks.keys():
            G.add_edge(reservoir, sink, weight=maxsize)
            costs[Edge((reservoir, sink))] = Cost(maxsize)
    if demand > 0:
        for src in sources.keys():
            G.add_edge(src, reservoir, weight=maxsize)
            costs[Edge((src, reservoir))] = Cost(maxsize)

    # Solve for the min-cost-flow
    solution_raw = nx.min_cost_flow(G)
    solution: Solution = Solution({})
    for fst_node, mapping in solution_raw.items():
        for snd_node, flow in mapping.items():
            if flow == 0 or fst_node == "RESERVOIR" or snd_node == "RESERVOIR":
                continue
            solution[Edge((fst_node, snd_node))] = AmountMoved(flow)

    # Calculate the cost
    accrued_costs: list[Cost] = [costs[e] for e in solution.keys()]
    avg_cost: AvgCost = AvgCost(sum(accrued_costs) / len(accrued_costs))

    # Print graph output
    if print_graph:
        # We can delete the reservoir since we're not using it anymore
        G.remove_node(reservoir)

        # Get the positions of each node
        pos = nx.multipartite_layout(G, subset_key="layer")

        # Format the colours for each node
        layer_colour = ["tab:red", "tab:blue"]
        node_colour = [layer_colour[data["layer"]] for _, data in G.nodes(data=True)]
        
        # Create the node and edge labels 
        #
        # Node labels show the offload and onload capacity
        node_labels: dict[Node, str] = {}
        for src, cap in sources.items():
            node_labels[src] = f"\n\n\n\n{src}\nTo Offload: {str(cap)}"
        for sink, cap in sinks.items():
            node_labels[sink] = f"\n\n\n\n{sink}\nCapacity: {str(cap)}"
        # Edge labels show how much of each flow we've moved
        sent_edges: list[Edge] = []
        edge_labels: dict[Edge, str] = {}
        for fst_node, mapping in solution_raw.items():
            for snd_node, flow_size in mapping.items():
                edge = Edge((Node(fst_node), Node(snd_node)))
                if fst_node == "RESERVOIR" or snd_node == "RESERVOIR":
                    continue
                if flow_size > 0:
                    sent_edges.append(edge)
                edge_labels[edge] = \
                        f"Cost: {str(costs[edge])}, Sent: {str(flow_size)}"
        
        # Draw the nodes and edges
        nx.draw(G, pos, node_color=node_colour)
        nx.draw_networkx_edges(G, pos,
                               width=3,
                               edgelist=sent_edges,
                               edge_color="tab:red",
                               )
        nx.draw_networkx_labels(G, pos, 
                                labels=node_labels,
                                font_size=10,
                                )
        nx.draw_networkx_edge_labels(G, pos,
                                     edge_labels=edge_labels,
                                     font_color='red',
                                     label_pos=0.7,
                                     )

        # Set the title
        plt.suptitle(f"{filename}\nflow moved: {total_moved(solution)}, mean cost: {avg_cost}")
        plt.margins(x=0.1, y=0.1, tight=True)

        # Save the graph
        plt.savefig(f"{filename}.png", format="PNG", dpi=300)

        # We need to clear the figure after each draw,
        # otherwise when we have multiple sequential 
        # calls we draw on the same figure
        plt.clf()

    return solution, avg_cost

