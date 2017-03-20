import random
import matplotlib.pyplot
import numpy
from Tkinter import *
import scipy.special

def zipf(a):
	if a <= 1:
		raise ValueError("The parameter should be greater than 1")
	a = float(a)
	b = (2 ** (a - 1))
	u = random.random()
	v = random.random()
	x = int(u ** (-1/(a-1)))
	t = (1 + 1/x) ** (a - 1)
	while v * x * ( (t - 1) / (b - 1)) > t / b :
		u = random.random()
		v = random.random()
		x = int(u ** (-1/(a-1)))
		t = (1 + 1/x) ** (a - 1)
	return x

def exp(a):
	return (scipy.special.zetac(a-1)+1)/(scipy.special.zetac(a)+1)

def prob(i, a):
	return 1.0 / ((scipy.special.zetac(a)+1) * (i ** a))

class MainWin:
	def __init__(self, master):
		frame = Frame(master)
		frame.pack()

		self.button = Button(frame, text="Inchide", fg="red", command=frame.quit)
		self.button.grid(row=1, column=2)

		self.hi_there = Button(frame, text="Calculeaza", command=self.get_stats)
		self.hi_there.grid(row=0, column=2)

		self.param_label = Label(frame, text="a: ")
		self.param_label.grid(row=0, column=0)

		self.param_entry = Entry(frame)
		self.param_entry.grid(row=0, column=1)

		self.avg1000_l = Label(frame, text="Medie (1000 de valori): ")
		self.avg1000_v = Entry(frame, state="readonly")
		self.avg1000_l.grid(row=1, column=0)
		self.avg1000_v.grid(row=1, column=1)

		self.avg10000_l = Label(frame, text="Medie (10000 de valori): ")
		self.avg10000_v = Entry(frame, state="readonly")
		self.avg10000_l.grid(row=2, column=0)
		self.avg10000_v.grid(row=2, column=1)

		self.avg100000_l = Label(frame, text="Medie (100000 de valori): ")
		self.avg100000_v = Entry(frame, state="readonly")
		self.avg100000_l.grid(row=3, column=0)
		self.avg100000_v.grid(row=3, column=1)

		self.avgexp_l = Label(frame, text="Medie teoretica: ")
		self.avgexp_v = Entry(frame, state="readonly")
		self.avgexp_l.grid(row=4, column=0)
		self.avgexp_v.grid(row=4, column=1)

	def get_stats(self):
		a = float(self.param_entry.get())
		l1000 = [zipf(a) for i in range(1000)]
		l10000 = [zipf(a) for i in range(10000)]
		l100000 = [zipf(a) for i in range(100000)]
		self.avg1000_v.config(state=NORMAL)
		self.avg10000_v.config(state=NORMAL)
		self.avg100000_v.config(state=NORMAL)
		self.avgexp_v.config(state=NORMAL)
		self.avg1000_v.delete(0, END)
		self.avg1000_v.insert(0, repr(numpy.mean(l1000, dtype=numpy.float64)))
		self.avg10000_v.delete(0, END)
		self.avg10000_v.insert(0, repr(numpy.mean(l10000, dtype=numpy.float64)))
		self.avg100000_v.delete(0, END)
		self.avg100000_v.insert(0, repr(numpy.mean(l100000, dtype=numpy.float64)))
		self.avgexp_v.delete(0, END)
		self.avgexp_v.insert(0, exp(a))
		self.avg1000_v.config(state="readonly")
		self.avg10000_v.config(state="readonly")
		self.avg100000_v.config(state="readonly")
		self.avgexp_v.config(state="readonly")
		self.show_hist(l100000, 1000, 6, a)

	def show_hist(self, l, n, k, a):
		ll = random.sample(l, n)
		bins = [0.0 for i in range(k+1)]
		bins[1] = min(ll)
		bins[k-1] = max(ll)
		bins[0] = min(l)
		bins[k] = max(l)
		h = (bins[k-1] - bins[1])/(k-2)
		for i in range(2, k-1):
			bins[i] = bins[i-1] + h
		x = []
		y = []
		for (i,b) in enumerate(bins[:-1]):
			if b != bins[i+1]:
				x.append(b)
				y.append(prob(b, a) * len(l)) # normalization so it looks good on the graph
		x.append(bins[k])
		y.append(prob(bins[k], a) * len(l))
		matplotlib.pyplot.clf()
		matplotlib.pyplot.hist(l, bins=bins, range=(bins[0], bins[k]), log=True)
		matplotlib.pyplot.plot(x, y, 'r--', drawstyle='line')
		matplotlib.pyplot.show()

def main():
	root = Tk()
	main_win = MainWin(root)
	root.mainloop()

if __name__ == '__main__':
	main()
 
