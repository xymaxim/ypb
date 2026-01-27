package input

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/oleiade/gomme"
)

const (
	NowKeyword      = "now"
	EarliestKeyword = "earliest"
)

const (
	OpPlus  = '+'
	OpMinus = '-'
)

// MomentValue defines the interface for all moment values.
type MomentValue any

type MomentExpression struct {
	Operator rune
	Left     MomentValue
	Right    time.Duration
}

type ParserResult = gomme.Result[MomentValue, string]

var intervalPart = gomme.Alternative(
	parseExpression,               // e.g., 2026-01-02T10:20:30+00 - 30s
	parseDateAndTime,              // e.g., 2026-01-02T10:20:30+00
	parseDuration,                 // e.g., 1d2h3m4s
	parseUnixTimestamp,            // e.g., @1767349230
	parseKeyword(NowKeyword),      // now
	parseKeyword(EarliestKeyword), // earliest
	parseSequenceNumber,           // e.g., 123
)

func ParseInterval(input string) (MomentValue, MomentValue, error) {
	result := gomme.SeparatedPair(
		intervalPart,
		gomme.Alternative(gomme.Token[string]("/"), gomme.Token[string]("--")),
		tillEnd(intervalPart),
	)(input)
	if result.Err != nil {
		return nil, nil, result.Err
	}

	start, end := result.Output.Left, result.Output.Right

	// Validate start value
	if start == NowKeyword {
		return nil, nil, fmt.Errorf(
			"keyword '%s' cannot be used as start",
			NowKeyword,
		)
	}

	// Validate end value
	if end == EarliestKeyword {
		return nil, nil, fmt.Errorf(
			"keyword '%s' cannot be used at end",
			EarliestKeyword,
		)
	}
	isDuration := func(v any) bool {
		_, ok := v.(time.Duration)
		return ok
	}
	if isDuration(start) && isDuration(end) {
		return nil, nil, errors.New("two durations are not allowed")
	}

	return start, end, nil
}

func ParseIntervalPart(input string) (MomentValue, error) {
	result := intervalPart(input)
	if result.Err != nil {
		return nil, result.Err
	}
	return result.Output, nil
}

func parseKeyword(keyword string) func(string) ParserResult {
	return func(input string) ParserResult {
		return gomme.Map(
			gomme.Terminated(
				gomme.Token[string](keyword),
				eof[string](),
			),
			func(keyword string) (MomentValue, error) {
				return keyword, nil
			},
		)(input)
	}
}

func parseSequenceNumber(input string) ParserResult {
	return gomme.Map(
		gomme.Digit1[string](),
		func(x string) (MomentValue, error) {
			return strconv.Atoi(x)
		},
	)(input)
}

func parseDateAndTime(input string) ParserResult {
	digits := func(n uint) gomme.Parser[string, int] {
		return gomme.Map(
			gomme.Take[string](n),
			strconv.Atoi,
		)
	}

	// Date parsers
	year := digits(4)
	month := gomme.Preceded(
		gomme.Char[string]('-'),
		digits(2),
	)
	day := gomme.Preceded(
		gomme.Char[string]('-'),
		digits(2),
	)

	dateOnly := gomme.Map(
		gomme.Sequence(year, month, day),
		func(parts []int) (MomentValue, error) {
			yyyy, mm, dd := parts[0], time.Month(parts[1]), parts[2]
			return time.Date(yyyy, mm, dd, 0, 0, 0, 0, time.UTC), nil
		},
	)

	// Time parsers
	hours := digits(2)
	minutes := gomme.Preceded(gomme.Char[string](':'), digits(2))
	seconds := gomme.Optional(gomme.Preceded(gomme.Char[string](':'), digits(2)))

	timeOnly := gomme.Map(
		gomme.Sequence(hours, minutes, seconds),
		func(parts []int) (MomentValue, error) {
			hh, mm, ss := parts[0], parts[1], parts[2]
			now := time.Now()
			return time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				hh,
				mm,
				ss,
				0,
				time.UTC,
			), nil
		},
	)

	// Offset parsers
	offsetHour := gomme.Map(
		gomme.Pair(gomme.OneOf[string]('+', '-'), digits(2)),
		func(p gomme.PairContainer[rune, int]) (int, error) {
			if p.Left == '-' {
				return -p.Right, nil
			}
			return p.Right, nil
		},
	)
	offsetMinutes := gomme.Optional(
		gomme.Preceded(
			gomme.Optional(gomme.Char[string](':')),
			digits(2),
		),
	)

	offset := gomme.Alternative(
		gomme.Map(
			gomme.Char[string]('Z'),
			func(_ rune) (*time.Location, error) {
				return time.UTC, nil
			},
		),
		gomme.Map(
			gomme.Sequence(offsetHour, offsetMinutes),
			func(parts []int) (*time.Location, error) {
				offsetHH, offsetMM := parts[0], parts[1]
				offsetSeconds := offsetHH*3600 + offsetMM*60
				return time.FixedZone(
					fmt.Sprintf("%+03d:%02d", offsetHH, offsetMM),
					offsetSeconds,
				), nil
			},
		),
	)

	withLocation := func(t time.Time, loc *time.Location) time.Time {
		newTime := t.In(loc)
		_, offsetSeconds := newTime.Zone()
		return newTime.Add(-time.Duration(offsetSeconds) * time.Second)
	}
	offsetted := func(
		t gomme.Parser[string, MomentValue],
	) gomme.Parser[string, MomentValue] {
		return gomme.Map(
			gomme.Pair(t, gomme.Optional(offset)),
			func(
				p gomme.PairContainer[MomentValue, *time.Location],
			) (MomentValue, error) {
				tt, loc := p.Left.(time.Time), p.Right
				if loc != nil {
					return withLocation(tt, loc), nil
				}
				return withLocation(tt, time.Local), nil //nolint:gosmopolitan
			},
		)
	}

	// All together
	all := offsetted(
		gomme.Alternative(
			gomme.Map(
				gomme.SeparatedPair(
					dateOnly,
					gomme.Char[string]('T'),
					timeOnly,
				),
				func(
					p gomme.PairContainer[MomentValue, MomentValue],
				) (MomentValue, error) {
					return time.Date(
						p.Left.(time.Time).Year(),
						p.Left.(time.Time).Month(),
						p.Left.(time.Time).Day(),
						p.Right.(time.Time).Hour(),
						p.Right.(time.Time).Minute(),
						p.Right.(time.Time).Second(),
						0,
						time.UTC,
					), nil
				},
			),
			dateOnly,
			timeOnly,
		),
	)

	return all(input)
}

func parseUnixTimestamp(input string) ParserResult {
	return gomme.Map(
		gomme.Preceded(
			gomme.Token[string]("@"),
			gomme.Int64[string](),
		),
		func(sec int64) (MomentValue, error) {
			return time.Unix(sec, 0).UTC(), nil
		},
	)(input)
}

func parseDuration(input string) ParserResult {
	dur := func(suffix rune) gomme.Parser[string, int] {
		return gomme.Optional(
			gomme.Terminated(
				integer[string](),
				gomme.Token[string](string(suffix)),
			),
		)
	}
	return gomme.Map(
		gomme.Sequence(dur('d'), dur('h'), dur('m'), dur('s')),
		func(parts []int) (MomentValue, error) {
			areAllEmpty := true
			for _, p := range parts {
				if p != 0 {
					areAllEmpty = false
				}
			}
			if areAllEmpty {
				return nil, gomme.NewError(input, "no any matches")
			}
			duration := time.Duration(parts[0])*time.Hour*24 +
				time.Duration(parts[1])*time.Hour +
				time.Duration(parts[2])*time.Minute +
				time.Duration(parts[3])*time.Second
			return duration, nil
		},
	)(input)
}

func parseExpression(input string) ParserResult {
	// Parse left operand
	leftResult := gomme.Terminated(
		gomme.Alternative(
			parseDateAndTime,
			parseUnixTimestamp,
			parseSequenceNumber,
			parseKeyword(NowKeyword),
		),
		gomme.Whitespace0[string](),
	)(input)
	if leftResult.Err != nil {
		return gomme.Failure[string, MomentValue](
			gomme.NewError(input, "parseExpression"),
			input,
		)
	}

	// Parse operator
	opResult := gomme.OneOf[string](OpPlus, OpMinus)(leftResult.Remaining)

	// Parse right operand
	rightResult := gomme.Preceded(
		gomme.Whitespace0[string](),
		parseDuration,
	)(opResult.Remaining)
	if rightResult.Err != nil {
		return gomme.Failure[string, MomentValue](
			gomme.NewError(input, "parseExpression"),
			input,
		)
	}

	return gomme.Result[MomentValue, string]{
		Output: MomentExpression{
			Left:     leftResult.Output,
			Operator: opResult.Output,
			Right:    rightResult.Output.(time.Duration),
		},
		Remaining: rightResult.Remaining,
	}
}

func eof[Input gomme.Bytes]() gomme.Parser[Input, Input] {
	return func(input Input) gomme.Result[Input, Input] {
		if len(input) == 0 {
			return gomme.Success(input, input)
		}
		return gomme.Failure[Input, Input](
			gomme.NewError(input, "end of input"),
			input,
		)
	}
}

func integer[Input gomme.Bytes]() gomme.Parser[Input, int] {
	return func(input Input) gomme.Result[int, Input] {
		parser := gomme.Recognize(gomme.Digit1[Input]())

		result := parser(input)
		if result.Err != nil {
			return gomme.Failure[Input, int](gomme.NewError(input, "integer"), input)
		}

		n, err := strconv.Atoi(string(result.Output))
		if err != nil {
			return gomme.Failure[Input, int](gomme.NewError(input, "integer"), input)
		}

		return gomme.Success(n, result.Remaining)
	}
}

func tillEnd[Input gomme.Bytes, Output any](
	parser gomme.Parser[Input, Output],
) gomme.Parser[Input, Output] {
	return func(input Input) gomme.Result[Output, Input] {
		result := parser(input)
		if result.Err != nil || len(result.Remaining) != 0 {
			return gomme.Failure[Input, Output](
				gomme.NewError(input, "tillEnd"),
				input,
			)
		}
		return gomme.Success(result.Output, result.Remaining)
	}
}
