package ansiterm

import (
	. "mayfly-go/internal/machine/mcm/ansiterm/consts"
)

var (
	Basic = map[string]struct{}{
		BEL: {},
		BS:  {},
		HT:  {},
		LF:  {},
		VT:  {},
		FF:  {},
		CR:  {},
		SO:  {},
		SI:  {},
	}
	Escape = map[string]struct{}{
		RIS:   {},
		IND:   {},
		NEL:   {},
		RI:    {},
		HTS:   {},
		DECSC: {},
		DECRC: {},
	}
	Sharp = map[string]struct{}{
		DECALN: {},
	}
	Csi = map[string]struct{}{
		ICH:     {},
		CUU:     {},
		CUD:     {},
		CUF:     {},
		HPR:     {},
		CUB:     {},
		CNL:     {},
		CPL:     {},
		CHA:     {},
		HPA:     {},
		CUP:     {},
		HVP:     {},
		ED:      {},
		EL:      {},
		IL:      {},
		DL:      {},
		DCH:     {},
		ECH:     {},
		DA:      {},
		DSR:     {},
		VPA:     {},
		TBC:     {},
		SM:      {},
		RM:      {},
		SGR:     {},
		DECSTBM: {},
	}
)

func (s *Stream) HandleBasic(char string) {
	switch char {
	case BEL:
		s.Listener.Bell()
	case BS:
		s.Listener.Backspace()
	case HT:
		s.Listener.Tab()
	case LF, VT, FF:
		s.Listener.LineFeed()
	case CR:
		s.Listener.CarriageReturn()
	case SO:
		s.Listener.ShiftOut()
	case SI:
		s.Listener.ShiftIn()
	}
}

func (s *Stream) HandleEscape(char string) {
	switch char {
	case RIS:
		s.Listener.Reset()
	case IND:
		s.Listener.Index()
	case NEL:
		s.Listener.LineFeed()
	case RI:
		s.Listener.ReverseIndex()
	case HTS:
		s.Listener.SetTabStop()
	case DECSC:
		s.Listener.SaveCursor()
	case DECRC:
		s.Listener.RestoreCursor()
	}

}

func (s *Stream) HandleSharp(char string) {
	switch char {
	case DECALN:
		s.Listener.AlignmentDisplay()
	}
}

func tidyParams(param []int, num int) []int {
	if len(param) < num {
		var newParams = make([]int, num)
		for i := 0; i < num; i++ {
			newParams[i] = 0
		}
		copy(newParams, param)
		return newParams
	}
	return param
}

func (s *Stream) HandleCSI(char string, params []int, kw map[string]any) {
	param := tidyParams(params, 1)
	switch char {
	case ICH:
		s.Listener.InsertCharacters(param[0])
	case CUU:
		s.Listener.CursorUp(param[0])
	case CUD:
		s.Listener.CursorDown(param[0])
	case CUF, HPR:
		s.Listener.CursorForward(param[0])
	case CUB:
		s.Listener.CursorBack(param[0])
	case CNL:
		s.Listener.CursorDown1(param[0])
	case CPL:
		s.Listener.CursorUp1(param[0])
	case CHA, HPA:
		s.Listener.CursorToColumn(param[0])
	case CUP, HVP:
		param = tidyParams(params, 2)
		s.Listener.CursorPosition(param[0], param[1])
	case ED:
		s.Listener.EraseInDisplay(param[0])
	case EL:
		if v, ok := kw["private"]; ok {
			switch v := v.(type) {
			case bool:
				s.Listener.EraseInLine(param[0], v)
				return
			}
		}
		s.Listener.EraseInLine(param[0], false)
	case IL:
		s.Listener.InsertLines(param[0])
	case DL:
		s.Listener.DeleteLines(param[0])
	case DCH:
		s.Listener.DeleteCharacters(param[0])
	case ECH:
		s.Listener.EraseCharacters(param[0])
	case DA, DSR:
		req := map[string]bool{}
		for k, v := range kw {
			switch v := v.(type) {
			case bool:
				req[k] = v
			}
		}
		s.Listener.ReportDeviceAttributes(param[0], req)
	case VPA:
		s.Listener.CursorToLine(param[0])
	case VPR:
		s.Listener.CursorDown(param[0])
	case TBC:
		s.Listener.ClearTabStop(param[0])
	case SM:
		s.Listener.SetMode(params, kw)
	case RM:
		s.Listener.ResetMode(params, kw)
	case SGR:
		s.Listener.SelectGraphicRendition(params...)
	case DECSTBM:
		s.Listener.SetMargins(param[0], param[1])
	default:
		// 未实现的 CSI 序列，静默忽略
	}
}
