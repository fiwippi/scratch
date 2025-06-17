package quantise

import (
	heapy "container/heap"
	"image"
	"image/color"
	"image/draw"
	"maps"
	"math"
	"slices"
	"sort"
)

// -- Node

type node struct {
	R, G, B     float64 // 0-255
	Prev        *node   // Pointer to the previous node
	Next        *node   // Pointer to the next node
	D           float64 // Merge cost value, indicating the increase in the MSE if the two classes are merged (this class and the one to the right)
	N           float64 // Number of pixels in the class
	Index       int     // Index of the node in the heap
	A           float64 // Alpha value of the node
	NN          *node   // Pointer to the nearest neighbour
	MergeCount  int     // The iteration where the node was last merged with another
	UpdateCount int     // The iteration where the MSE was last calculated for the node
}

func sqr(a float64) float64 {
	return a * a
}

func (n *node) nearestNeighbour() {
	var err = math.MaxFloat64
	var nn *node

	next := n.Next
	for next != nil {
		nerr := vectorCost(n, next)
		if nerr < err {
			err = nerr
			nn = next
		}
		next = next.Next
	}

	n.NN = nn
	n.D = err
}

// Calculates the cost of merging two colour clusters,
// it represents the increase in MSE value caused by
// the merge
func vectorCost(a, b *node) float64 {
	lhs := (a.N * b.N) / (a.N + b.N)
	rhs := sqr(b.A-a.A) + sqr(b.R-a.R) + sqr(b.G-a.G) + sqr(b.B-a.B)

	return lhs * rhs
}

// -- Heap

type heap []*node

func (h heap) Len() int {
	return len(h)
}

func (h heap) Less(i, j int) bool {
	// We want Pop to give us the node with lowest cost so we use "<" here.
	return h[i].D < h[j].D
}

func (h heap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *heap) Push(x interface{}) {
	n := len(*h)
	item := x.(*node)
	item.Index = n
	*h = append(*h, item)
}

func (h *heap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	item.Index = -1 // For safety
	*h = old[0 : n-1]
	return item
}

func (h *heap) Front() interface{} {
	return (*h)[0]
}

func (h *heap) Update(node *node, d float64) {
	node.D = d
	heapy.Fix(h, node.Index)
}

func (h *heap) RecalculateNeighbours(count int) *node {
	for {
		S := h.Front().(*node)

		if S.UpdateCount >= S.MergeCount && S.UpdateCount >= S.NN.MergeCount {
			return S
		} else {
			S.nearestNeighbour()
			heapy.Fix(h, S.Index)
			S.UpdateCount = count
		}
	}
}

// -- Histogram

type histogram map[uint32]*node

// Takes uint32 RGBA colours and gives them a unique uint16 index value,
// this simplifies the colour space and speeds up computation without
// a noticeable loss in quality
func argbIndex(a, r, g, b uint32) uint32 {
	return (a&0xF0)<<8 | (r&0xF0)<<4 | (g & 0xF0) | (b >> 4)
}

func newHistogram(img image.Image) histogram {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	pixels := make(histogram)
	for y := bounds.Min.Y; y < height; y++ {
		for x := bounds.Min.X; x < width; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			// Convert rgb values to be in range 0-255 so 8 bits for grayscale
			a, r, g, b = a>>8, r>>8, g>>8, b>>8

			// Get a uint16 number to use as an index for the colour
			index := argbIndex(a, r, g, b)

			// Create a node if it doesnt exist
			if pixels[index] == nil {
				pixels[index] = &node{}
			}

			// Add the pixel to the bin
			pixels[index].A += float64(a)
			pixels[index].R += float64(r)
			pixels[index].G += float64(g)
			pixels[index].B += float64(b)
			pixels[index].N++
		}
	}

	return pixels
}

func (hist histogram) initialiseColours() (*node, *heap) {
	var (
		currentNode  *node
		previousNode *node
		keys         = slices.Sorted(maps.Keys(hist))
		head         = hist[keys[0]]
	)

	for _, i := range keys {
		currentNode = hist[i]
		currentNode.A /= currentNode.N
		currentNode.R /= currentNode.N
		currentNode.G /= currentNode.N
		currentNode.B /= currentNode.N

		currentNode.Prev = previousNode
		if previousNode != nil {
			previousNode.Next = currentNode
		}

		previousNode = currentNode
	}

	h := make(heap, 0)
	heapy.Init(&h)

	n := head
	for n != nil {
		n.nearestNeighbour()
		if n.Next != nil {
			heapy.Push(&h, n)
		}
		n = n.Next
	}

	return head, &h
}

// PNN quantisation, taken from Virmajoki, O., & Franti, P. (2003). Multilevel
// thresholding by fast PNN-based algorithm. Image Processing: Algorithms and
// Systems II.

func Quantise(img image.Image, size int) color.Palette {
	S, H := newHistogram(img).initialiseColours()

	m := H.Len() + 1
	count := 0
	for m != size {
		n := H.RecalculateNeighbours(count)
		updateQuantiserState(n, n.NN, H, count)

		m = m - 1
		count += 1
	}

	thresholds := make(color.Palette, 0, size)
	for S != nil {
		clr := color.RGBA{uint8(S.R), uint8(S.G), uint8(S.B), uint8(S.A)}
		thresholds = append(thresholds, clr)
		S = S.Next
	}

	return thresholds
}

func updateQuantiserState(a, b *node, H *heap, count int) {
	Nq := a.N + b.N
	a.A = (a.N*a.A + b.N*b.A) / Nq
	a.R = (a.N*a.R + b.N*b.R) / Nq
	a.G = (a.N*a.G + b.N*b.G) / Nq
	a.B = (a.N*a.B + b.N*b.B) / Nq
	a.N = Nq

	// Unchain the nearest neighbour bin
	if b.Next != nil {
		b.Next.Prev = b.Prev
	}
	if b.Prev != nil {
		b.Prev.Next = b.Next
	}

	// Remove the neighbour from the bin
	if b.Index >= 0 && b.Index < H.Len() {
		_ = heapy.Remove(H, b.Index)
	}

	// Remove element from heap if its at the
	// end of the list and not already removed
	if a.Next == nil && a.Index != -1 {
		_ = heapy.Remove(H, a.Index)
	}

	a.MergeCount = count + 1
	b.MergeCount = math.MaxInt32
}

func Palette(p color.Palette, size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size*len(p), size))

	for i, v := range p {
		uniformColour := image.NewUniform(v)
		draw.Draw(img, image.Rect(size*i, 0, size*i+size, size), uniformColour, image.Point{}, draw.Src)
	}

	return img
}
