package iso8583

// FieldCodec enumerates encoding formats for ISO8583 data elements.
type FieldCodec int

const (
	FmtFixedNum FieldCodec = iota // ASCII numeric fixed
	FmtFixedAns                   // ASCII ans fixed
	FmtLLVAR                      // ASCII ans LLVAR
	FmtLLLVAR                     // ASCII ans LLLVAR
)

// FieldSpec describes an ISO8583 data element.
type FieldSpec struct {
	Num   int
	Name  string
	Codec FieldCodec
	Len   int // length for fixed fields
}

// CommonSpec lists common ISO8583 fields supported by this package.
var CommonSpec = map[int]FieldSpec{
	2:   {2, "PAN", FmtLLVAR, 0},
	3:   {3, "ProcessingCode", FmtFixedNum, 6},
	4:   {4, "Amount", FmtFixedNum, 12},
	7:   {7, "TransmissionDateTime", FmtFixedNum, 10},
	11:  {11, "STAN", FmtFixedNum, 6},
	12:  {12, "LocalTime", FmtFixedNum, 6},
	13:  {13, "LocalDate", FmtFixedNum, 4},
	14:  {14, "Expiry", FmtFixedNum, 4},
	22:  {22, "POSEntryMode", FmtFixedNum, 3},
	23:  {23, "PANSeq", FmtFixedNum, 3},
	24:  {24, "NII", FmtFixedNum, 3},
	25:  {25, "POSCond", FmtFixedNum, 2},
	32:  {32, "AcqInstID", FmtLLVAR, 0},
	35:  {35, "Track2", FmtLLVAR, 0},
	37:  {37, "RRN", FmtFixedAns, 12},
	38:  {38, "AuthID", FmtFixedAns, 6},
	39:  {39, "RespCode", FmtFixedAns, 2},
	41:  {41, "TermID", FmtFixedAns, 8},
	42:  {42, "MerchID", FmtFixedAns, 15},
	43:  {43, "MerchLoc", FmtFixedAns, 40},
	48:  {48, "AddlDataPriv", FmtLLLVAR, 0},
	49:  {49, "Currency", FmtFixedAns, 3},
	52:  {52, "PINBlock", FmtFixedAns, 16},
	53:  {53, "SecCtrl", FmtFixedNum, 16},
	54:  {54, "AddlAmounts", FmtLLLVAR, 0},
	55:  {55, "ICCData", FmtLLLVAR, 0},
	60:  {60, "AdviceReason/Priv", FmtLLLVAR, 0},
	61:  {61, "POSExt", FmtLLLVAR, 0},
	62:  {62, "Priv", FmtLLLVAR, 0},
	63:  {63, "Priv2", FmtLLLVAR, 0},
	70:  {70, "NMMCode", FmtFixedNum, 3},
	102: {102, "AccountID1", FmtLLVAR, 0},
}
