all: gofmt data/fraglibs/centers400_11

install:
	go install -p 6 ./fragbag ./matt ./pdb ./rmsd
	go install -p 6 ./cmd/*

gofmt:
	gofmt -w */*.go cmd/*/*.go */example/*/*.go
	colcheck */*.go cmd/*/*.go */example/*/*.go

tags:
	find ./ \( \
			-name '*.go' \
			-and -not -wholename './examples/*' \
		\) -print0 \
		| xargs -0 gotags > TAGS

loc:
	find ./ -name '*.go' -print | sort | xargs wc -l

data/fraglibs/%: data/fraglibs/%.brk
	scripts/translate-fraglib "data/fraglibs/$*.brk" "data/fraglibs/$*"

