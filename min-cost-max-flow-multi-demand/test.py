import unittest
from random import randint
from time import perf_counter

from flow import *

class TestFlowsMoved(unittest.TestCase):
    def test_choose_closest_single(self):
        sources: Nodes = Nodes({
                Node("A"): Capacity(1),
                Node("B"): Capacity(1),
                })
        sinks: Nodes = Nodes({
                Node("C"): Capacity(0),
                Node("D"): Capacity(1),
                })
        costs: Edges = Edges({
                Edge((Node("A"), Node("C"))): Cost(10),
                Edge((Node("A"), Node("D"))): Cost(15),
                Edge((Node("B"), Node("C"))): Cost(10),
                Edge((Node("B"), Node("D"))): Cost(10),
                })
     
        name = self.id().split(".")[-1]
        s, cost = match_flows(sources, sinks, costs, filename=name)
        self.assertEqual(total_moved(s), AmountMoved(1))
        self.assertEqual(cost, AvgCost(10.0))


    def test_choose_closest_multi(self):
        sources: Nodes = Nodes({
                Node("A"): Capacity(1),
                Node("B"): Capacity(1),
                })
        sinks: Nodes = Nodes({
                Node("C"): Capacity(1),
                Node("D"): Capacity(1),
                })
        costs: Edges = Edges({
                Edge((Node("A"), Node("C"))): Cost(10),
                Edge((Node("A"), Node("D"))): Cost(15),
                Edge((Node("B"), Node("C"))): Cost(10),
                Edge((Node("B"), Node("D"))): Cost(10),
                })
     
        name = self.id().split(".")[-1]
        s, cost = match_flows(sources, sinks, costs, filename=name)
        self.assertEqual(total_moved(s), AmountMoved(2))
        self.assertEqual(cost, AvgCost(10.0))
    
    def test_choose_furthest_single(self):
        sources: Nodes = Nodes({
                Node("A"): Capacity(1),
                Node("B"): Capacity(1),
                })
        sinks: Nodes = Nodes({
                Node("C"): Capacity(0),
                Node("D"): Capacity(1),
                })
        costs: Edges = Edges({
                Edge((Node("A"), Node("C"))): Cost(10),
                Edge((Node("A"), Node("D"))): Cost(10),
                Edge((Node("B"), Node("C"))): Cost(10),
                Edge((Node("B"), Node("D"))): Cost(15),
                })
     
        name = self.id().split(".")[-1]
        s, cost = match_flows(sources, sinks, costs, filename=name)
        self.assertEqual(total_moved(s), AmountMoved(1))
        self.assertEqual(cost, AvgCost(10.0))

    def test_choose_furthest_multi(self):
        sources: Nodes = Nodes({
                Node("A"): Capacity(1),
                Node("B"): Capacity(1),
                })
        sinks: Nodes = Nodes({
                Node("C"): Capacity(1),
                Node("D"): Capacity(1),
                })
        costs: Edges = Edges({
                Edge((Node("A"), Node("C"))): Cost(10),
                Edge((Node("A"), Node("D"))): Cost(15),
                Edge((Node("B"), Node("C"))): Cost(2),
                Edge((Node("B"), Node("D"))): Cost(10),
                })
     
        name = self.id().split(".")[-1]
        s, cost = match_flows(sources, sinks, costs, filename=name)
        self.assertEqual(total_moved(s), AmountMoved(2))
        self.assertEqual(cost, AvgCost(8.5))

    def test_choose_furthest_multi_v2(self):
        sources: Nodes = Nodes({
                Node("A"): Capacity(1),
                Node("B"): Capacity(1),
                Node("C"): Capacity(4),
                })
        sinks: Nodes = Nodes({
                Node("D"): Capacity(3),
                Node("E"): Capacity(3),
                })
        costs: Edges = Edges({
                Edge((Node("A"), Node("D"))): Cost(10),
                Edge((Node("A"), Node("E"))): Cost(15),
                Edge((Node("B"), Node("D"))): Cost(3),
                Edge((Node("B"), Node("E"))): Cost(10),
                Edge((Node("C"), Node("D"))): Cost(5),
                Edge((Node("C"), Node("E"))): Cost(10),
                })
     
        name = self.id().split(".")[-1]
        s, cost = match_flows(sources, sinks, costs, filename=name)
        self.assertEqual(total_moved(s), AmountMoved(6))
        self.assertEqual(cost, AvgCost(8.25))

    def test_long(self):
        nsrc = 1000
        nsnk = 1000

        sources: Nodes = {}
        for i in range(nsrc):
            sources[Node(f"{i}-src")] = Capacity(randint(1,100))

        sinks: Nodes = {}
        for i in range(nsnk):
            sinks[Node(f"{i}-snk")] = Capacity(randint(1,100))

        costs: Edges = {}
        for i in range (nsrc):
            for j in range (nsnk):
                src = Node(f"{i}-src")
                sink = Node(f"{j}-snk")
                costs[Edge((src, sink))] = Cost(randint(1, 3000))

        start = perf_counter()
        sol, cost = match_flows(sources, sinks, costs)
        print(f"test_long: {perf_counter()-start}")


if __name__ == '__main__':
    unittest.main()

