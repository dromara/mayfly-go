package ansiterm

type Savepoint struct {
	Cursor    Cursor
	G0Charset map[rune]rune
	G1Charset map[rune]rune
	Charset   int
	Origin    bool
	Wrap      bool
}

type Margins struct {
	Top    int
	Bottom int
}

type Char struct {
	Data          string
	Fg            string
	Bg            string
	Bold          bool
	Italics       bool
	Underscore    bool
	Strikethrough bool
	Reverse       bool
	Blink         bool
}

func (c *Char) Update(data map[string]any) {
	for k, v := range data {
		switch k {
		case "data":
			c.Data = v.(string)
		case "fg":
			c.Fg = v.(string)
		case "bg":
			c.Bg = v.(string)
		case "bold":
			c.Bold = v.(bool)
		case "italics":
			c.Italics = v.(bool)
		case "underscore":
			c.Underscore = v.(bool)
		case "strikethrough":
			c.Strikethrough = v.(bool)
		case "reverse":
			c.Reverse = v.(bool)
		case "blink":
			c.Blink = v.(bool)
		}
	}
}
func (c *Char) Map() (data map[string]any) {
	data = map[string]any{
		"data":          c.Data,
		"fg":            c.Fg,
		"bg":            c.Bg,
		"bold":          c.Bold,
		"italics":       c.Italics,
		"underscore":    c.Underscore,
		"strikethrough": c.Strikethrough,
		"reverse":       c.Reverse,
		"blink":         c.Blink,
	}
	return
}

func NewChar(data string) Char {
	return Char{
		Data: data,
		Fg:   "default",
		Bg:   "default",
	}
}

type Cursor struct {
	X      int
	Y      int
	Attrs  Char
	Hidden bool
}

// NewCursor creates a new Cursor with the specified position and attributes.
func NewCursor(x int, y int, attrs Char) Cursor {
	return Cursor{
		X:     x,
		Y:     y,
		Attrs: attrs,
	}
}

type StaticDefaultDict[KT comparable, VT any] struct {
	Data       map[KT]VT
	DefaultVal VT
}

// NewStaticDefaultDict creates a new StaticDefaultDict with the provided default value.
func NewStaticDefaultDict[KT comparable, VT any](defaultVal VT) *StaticDefaultDict[KT, VT] {
	return &StaticDefaultDict[KT, VT]{
		Data:       make(map[KT]VT),
		DefaultVal: defaultVal,
	}
}

func (sd *StaticDefaultDict[KT, VT]) Get(key KT) VT {
	value, exists := sd.Data[key]

	if !exists {
		return sd.DefaultVal
	}
	return value
}

func (sd *StaticDefaultDict[KT, VT]) Set(key KT, value VT) {
	if sd.Data == nil {
		sd.Data = map[KT]VT{
			key: value,
		}
	} else {
		sd.Data[key] = value
	}
}

func (sd *StaticDefaultDict[KT, VT]) Del(key KT) {
	//sd.data[key] = value
	delete(sd.Data, key)
}

type ScreenBuffer struct {
	Map         map[int]*StaticDefaultDict[int, Char]
	DefaultChar Char
}

func NewScreenBuffer(defaultVal Char) *ScreenBuffer {
	return &ScreenBuffer{
		Map:         make(map[int]*StaticDefaultDict[int, Char]),
		DefaultChar: defaultVal,
	}
}

func (db *ScreenBuffer) Get(key int) *StaticDefaultDict[int, Char] {
	if sdd, exists := db.Map[key]; exists {
		return sdd
	}
	db.Map[key] = &StaticDefaultDict[int, Char]{
		Data:       make(map[int]Char),
		DefaultVal: db.DefaultChar,
	}
	return db.Map[key]
}

func (db *ScreenBuffer) GetValue(key int) StaticDefaultDict[int, Char] {
	v := &StaticDefaultDict[int, Char]{
		Data:       make(map[int]Char),
		DefaultVal: db.DefaultChar,
	}
	if sdd, exists := db.Map[key]; exists {
		v = sdd
	} else {
		v = &StaticDefaultDict[int, Char]{
			Data:       make(map[int]Char),
			DefaultVal: db.DefaultChar,
		}
		db.Map[key] = v
	}
	return *v
}

func (db *ScreenBuffer) Set(key int, value StaticDefaultDict[int, Char]) {
	db.Map[key] = &value
}

func (db *ScreenBuffer) Delete(key int) {
	delete(db.Map, key)
}

func (db *ScreenBuffer) Clear() {
	db.Map = make(map[int]*StaticDefaultDict[int, Char])
}

func (db *ScreenBuffer) HasKey(key int) bool {
	if _, ok := db.Map[key]; ok {
		return ok
	}
	return false
}
