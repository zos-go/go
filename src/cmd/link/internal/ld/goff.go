// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ld

import (
	"bytes"
	"cmd/internal/obj"
	"cmd/internal/obj/s390x"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
	"unsafe"
)

///////////////////////////////////////////////////////////////////////
///
/// ZOS particular object declarations for:
///
///    (1) GOFF object file logical ESD, TXT, RLD and other types.
///
///    (2) GOFF object file concrete
///        ESD, TXT, RLD and other types
///
///    (3) Control blocks like CEESTART
///
///////////////////////////////////////////////////////////////////////

//
//
// (1) Logical ESD, TXT, RLD types follow
//     These logical types apply to both GOFF and
//
//

// Logical ESD types
// For GOFF these logical types are mapped to ESDTY_XX values
const (
	EXT_SD = 1 //   section definition
	// SD is used to define a new code/data section.
	// In GOFF, it has no parent and there will always be an associated ED created, and an
	// associated LD if it may be referenced externally.
	// Note: The SD name may be blank.

	EXT_ED = 2 //   element definition
	// ED is used to provide the characteristics of a section.  Its parent is the owning SD.
	// The exi_dotdrx provides its class name.

	EXT_LD = 3 //   label definition
	// LD is used to define an externally visible label (e.g. entrypoint).  In GOFF, its
	// parent is the owning ED.

	EXT_ER = 4 //   external reference
	// ER is used to indicate an  external reference.  In GOFF, its parent is the owning SD,
	// or zero if it has no parent.
	// Note: The exf_weak_ref flag may be set to indicate a weak external reference.

	EXT_PR = 5 //   pseudo-register/part reference
	// PR is used to initialize or to reference variables that are contained in the WSA.  In
	// GOFF, its parent is the owning ED.
)

// for exi_entry (ESD index)
const (
	EXINS_CODE = 1 // These values are used for namespace
	EXINS_PR   = 2
	EXINS_DATA = 3

	EXIAL_BYTE   = 0 //   N.B.: These values are used
	EXIAL_HWORD  = 1 //         as-is for the ESD alignment
	EXIAL_FWORD  = 2 //         value in GOFF format.
	EXIAL_DWORD  = 3
	EXIAL_QWORD  = 4
	EXIAL_PAGE   = 12
	EXIAL_DSPACE = 31

	// priorities to be assigned to WSA PR items
	STAT_INIT_ESD_PRIORITY = 1000
	PTR_INIT_ESD_PRIORITY  = 2000
	NZ_INIT_ESD_PRIORITY   = 3000
	Z_INIT_ESD_PRIORITY    = 4000
	N_INIT_ESD_PRIORITY    = 5000
)

type exi_entry struct {
	exi_name      string // const char in C
	exi_namespace uint32

	exi_offset    uint32 // offset for SD,LD
	exi_length    uint32 // total length of section
	exi_alignment uint32 // alignment for ED,PR:

	exi_first_txi_ix    uint32 // first txi_entry() index (SD)
	exi_last_txi_ix     uint32 // last  txi_entry() index (SD)
	exi_att_count       uint32 // count of ATT's for this exi
	exi_att_offset      uint32 // offs of ext attr hdr in C_EXTNATTR
	exi_first_att_ix    uint32 // first att_entry() index
	exi_last_att_ix     uint32 // last  att_entry() index
	exi_sort_priority   uint32 // sort priority (PR)
	exi_type            uint32 // external type:
	exi_esdid           uint32 // its ESDid
	exi_parent_exi_ix   uint32 // its parent's exi_ix
	exf_force_rent      bool   // GOFF: force this as RENT
	exf_associated      bool   // XPlink: "Associated"
	exf_xplink          bool   // Mark this def/ref as XPLink
	exf_deferred        bool   // GOFF mode, deferred load class
	exf_noload          bool   // GOFF mode, noload load class
	exf_indirect        bool   // indirect reference
	exf_use_sname       bool   // exi_sname has short name
	exf_weak_def        bool   // LD weak external definition
	exf_weak_ref        bool   // ER weak external reference
	exf_mangled         bool   // name mangled
	exf_internal        bool   // internal function
	exf_removable       bool   // GOFF: is_removable
	exf_merge           bool   // merge/concat
	exf_export          bool   // exported
	exf_executable      bool   // it's code (else data)
	exf_mapped          bool   // name is mapped
	exf_c_wsa           bool   // this is for C_WSA
	exf_c_private_wsa   bool   // this is for C_WSA24
	exf_def             bool   // ext_def
	exf_readonly        bool   // is read only
	exf_execunspecified bool
	exi_sname           [8]uint8 // short name, if exf_use_sname=TRUE
	exi_flag            uint8    // text record style
	// exi_rec_style:4
}

// For rli_entry
const (
	RLT_LOW      = 1  //   (lowest value number)
	RLT_ADCON    = 1  //   Adcon
	RLT_VCON     = 2  //   Vcon
	RLT_QCON     = 3  //   Qcon
	RLT_LEN      = 4  //   Length
	RLT_ADA      = 5  //   ADA for an XPLink entrypoint
	RLT_RI       = 6  //   RI relative immed
	RLT_REL      = 7  //   relative
	RLT_LD       = 20 //   the new GOFF Long Disp reloc
	RLT_CONDVCON = 21 //   the new GOFF conditional VCON
	// chwan - as marked below RLT_VCON does not work
	RLT_XVCON = 22 //   special VCON for an XPLink EP

	RS_NONE = 0  //   no relocation; ignore content
	RS_POS  = 1  //   positive
	RS_NEG  = -1 //   negative
	RS_ADA  = 2  //   ADA for an XPLink entrypoint
	RS_RILP = 3  //   relative immediate long positive
	RS_RILN = 4  //   relative immediate long negative
	// chwan - to trigger the use of RLT_XVCON
	RS_EP = 5 //   EP for an XPLink entrypoint
)

type rli_entry struct {
	rli_next_in_part uint32 // ix of next in part (or 0 if last)
	rli_ref_exi_ix   uint32 // index of exi being referenced
	rli_in_exi_ix    uint32 // index of exi containing adcon
	rli_offset       int32  // offset of adcon within rli_inexi
	rli_length       int32  // adcon length
	rli_type         int16  // relocation type:
	rli_sign         int16  // sign of relocation for RLT_ADCON:
}

type txi_entry struct {
	txi_next_ix  uint32  // txi_ix of next TXI for the same SD
	txi_exi_ix   uint32  // associated exi_entry() index
	txi_offset   int32   // starting offset of text data
	txi_length   uint32  // length of text data
	txi_data_ptr *[]byte // -> text data
	txi_flag     uint8   // flag:
	// txi_rec_style:4
}

/*
 * GOFF definitions that are independent of architecture or word size.
 */

//
//
// (2) GOFF
//
// The following describing GOFF record layouts.
// The data layout below was forced to fit into 80-byte records.

// Lines marked with +++ in their comments indicate additions per the
// "Program Management Support for High-Performance Linkage, Ver 1.2.1"
// document (by Leona Baumgart) for XPLink and/or 64-bit support, and its
// follow-ons.

const (
	GOFF_PTV_PREFIX = 0x03
	GOFF_ESD        = 0
	GOFF_TXT        = 1
	GOFF_RLD        = 2
	GOFF_LEN        = 3
	GOFF_END        = 4
	GOFF_HDR        = 15
)

/*
 * GOFF header record.
 * Note: It's a fixed length.
 */
const (
	HDR_MOD_LEN   = 4
	HDR_SYSTEM_LE = 0x8000
)

type goff_hdr_record struct {
	// hdr_fixed_part
	hdr_ptv_prefix uint8

	hdr_ptv_flag uint8 // flag:
	//   hdr_ptv_type:4;
	//   hdr_ptv_reserved:2;
	//   hdr_ptv_continuation:1;  - continuation of previous record
	//   hdr_ptv_continued:1;     - continuated on to the next record

	hdr_ptv_version uint8 // always 0
	hdr_reserved_1  uint8 // reserved

	// @@@ Start: The setting of these fields are yet to be finalized
	hdr_hardware_env  int32 // target hardware environment
	hdr_os            int32 // target operating system
	hdr_CCSID         int32 // CCSID for external symbols
	hdr_char_set_name [16]uint8
	hdr_lang_prod_id  [16]uint8 // compiler language

	// @@@ End:
	// @@@ we are setting level=1 which is for PM4 targetting V2R10. May be ok.
	hdr_arch_level         int32 // GOFF arch level  153060
	hdr_mod_properties_len int16 // len of mod properties field

	//HDR_MOD_LEN 4
	hdr_reserved_2 [6]uint8 // reserved

	// @@@ In order to set hdr_software_env, hdr_internal_CCSID has to be
	//     included. We'll leave it alone for now.
	hdr_internal_CCSID uint16    // CCSID for internal strings
	hdr_software_env   uint16    // target software environment
	hdr_reserved_3     [16]uint8 // 80-byte record filler
}

// GOFF ESD record.
// Note: Some fields apply only for certain ESD types; these are
//       indicated (in parenthesis) in the comments.
// Note: There's a fixed header portion and a variable name portion.
// Note: The blank line spaces below are used to designate byte
//       boundaries.
const (
	ESDTY_SD = 0
	ESDTY_ED = 1
	ESDTY_LD = 2
	ESDTY_PR = 3
	ESDTY_ER = 4

	ESDES_NONE    = 0
	ESDES_SECTION = 1
	ESDES_LABEL   = 2
	ESDES_CLASS   = 3
	ESDES_PART    = 4

	// ??  esd_ed_reserve_qwords = esd_er_symbol_type

	ESDRQ_0 = 0 //   0 qwords (0 bytes)
	ESDRQ_1 = 1 //   1 qword  (16 bytes)
	ESDRQ_2 = 2 //   2 qwords (32 bytes)
	ESDRQ_3 = 3 //   ...and so on...

	AMODE_NONE     = 0
	AMODE_24       = 1
	AMODE_31       = 2
	AMODE_24_OR_31 = 3 // (Better name for ANY)
	AMODE_64       = 4
	AMODE_MIN      = 16

	RMODE_NONE = 0
	RMODE_24   = 1
	RMODE_31   = 3
	RMODE_64   = 4

	ESDTS_BYTE         = 0
	ESDTS_STRUCTURED   = 1
	ESDTS_UNSTRUCTURED = 2

	ESDBA_CONCAT = 0
	ESDBA_MERGE  = 1

	ESDTA_UNSPECIFIED = 0
	ESDTA_NON_REUS    = 1
	ESDTA_REUS        = 2
	ESDTA_RENT        = 3

	ESDEX_UNSPECIFIED = 0
	ESDEX_DATA        = 1
	ESDEX_INSTR       = 2

	MDEF_NO_WARNING = 0
	MDEF_WARNING    = 1
	MDEF_ERROR      = 2
	MDEF_SEV_ERROR  = 3

	ESDST_STRONG = 0
	ESDST_WEAK   = 1

	ESDCL_INITIAL  = 0
	ESDCL_DEFERRED = 1
	ESDCL_NOLOAD   = 2

	ESDSC_UNSPECIFIED   = 0
	ESDSC_SECTION       = 1
	ESDSC_MODULE        = 2
	ESDSC_LIBRARY       = 3
	ESDSC_EXPORT_IMPORT = 4
)

type goff_esd_record struct {

	// esd_fixed_part
	esd_ptv_prefix uint8

	esd_ptv_flag1 uint8 // flag1:
	//   esd_ptv_type:4    uint8
	//   esd_ptv_reserved:2;
	//   esd_ptv_continuation:1; - continuation of previous record
	//   esd_ptv_continued:1;    - continuated on to the next record

	esd_ptv_version uint8 // always 0
	esd_symbol_type uint8 // symbol type:

	esd_esdid           int32
	esd_parent_esdid    int32 // (parent of SD is 0)
	esd_reserved_1      int32 // reserved for 64 bit...
	esd_offset          int32 // (LD)
	esd_reserved_2      int32 // reserved for 64 bit...
	esd_length          int32 // (ED,PR)
	esd_ext_attr_esdid  int32 // (LD,ER) 0 if none
	esd_ext_attr_offset int32 // (LD,ER)
	esd_alias           int32 // (LD,ER)

	esd_name_space_id uint8 // (ED,LD,PR,ER)

	esd_ptv_flag2 uint8 // flag2:
	//   esd_fill_byte_present:1   // (SD,ED,PR)
	//   esd_name_mangled:1;
	//   esd_sym_renamable:1;      // (LD,PR,ER)
	//   esd_removable:1;          // 310642
	//   esd_reserved_3:1;         // reserved
	//   esd_er_symbol_type:3;     // type of symbol (ER):

	//    uint8_t           esd_ed_reserve_qwords:3;  // reserve # qwords (ED):
	esd_fill_byte_value uint8 // (SD,ED,PR)
	esd_reserved_4      uint8 // reserved

	esd_ada_esdid     int32    // esdid of ADA (LD)  +++
	esd_sort_priority int32    // sort priority (PR)
	esd_signature     [8]uint8 // signature (LD,ER)

	// esd_behavioural_attr
	esd_amode uint8 // amode (ED,LD,ER):
	esd_rmode uint8 // rmode (ED):

	esd_ptv_flag3 uint8 // flag3:
	//	esd_text_rec_style:4      // text record style (ED):
	//	esd_binding_algorithm:4;  // binding alg'm (ED,LD):

	esd_ptv_flag4 uint8 // flag4:
	//	esd_tasking_behaviour:3   // tasking behaviour (SD):
	//	esd_movable:1;            // movable flag (ED)
	//	esd_read_only:1;          // read-only flag (ED)
	//	esd_executable:3;         // executable(ED,LD,ER,PR)

	esd_ptv_flag5 uint8 // flag5:
	//	esd_must_conform:1 uint8       // XPLink ref match def+++
	//	esd_associate_ada_epa:1;  // XPLink assoc ADA/EPA+++
	//	esd_mdef:2;
	//	esd_binding_strength:4;   // bind strength (LD,ER):

	esd_ptv_flag6 uint8 // flag6:
	//	esd_loading_behaviour:2  uint8   // loading behaviour (ED):
	//	esd_common:1;             // common flag (SD)
	//	esd_indirect_reference:1; // indirect ref (PR,ER)
	//	esd_binding_scope:4;      // bind scope (LD,PR,ER):

	esd_ptv_flag7 uint8 // flag7:
	//	esd_reserved_5:2        uint8         // (reserved)         +++
	//	esd_linkage_xplink:1;     // XPLink (LD,PR,ER): +++
	// N.B.: The EXIAL_xxxx literal values are used here
	//       directly, so they MUST be valid GOFF values.
	//	esd_alignment:5;          // alignment (ED,PR):

	esd_reserved_6  [3]uint8
	esd_name_length int16 // total len of esd_name

	// And now the variable part...
	esd_name string
}

// GOFF TXT record.
// Note: There's a fixed header portion and a variable data portion.
const (
	TXTTS_BYTE         = 0
	TXTTS_STRUCTURED   = 1
	TXTTS_UNSTRUCTURED = 2
)

type goff_txt_record struct {
	// txt_fixed_part
	txt_ptv_prefix uint8

	txt_ptv_flag1 uint8 // flag1:
	//	txt_ptv_type:4  uint8
	//	txt_ptv_reserved:2;
	// 	txt_ptv_continuation:1;   // continuation of previous record
	//	txt_ptv_continued:1;      // continuated on to the next record

	txt_ptv_version uint8 // always 0

	txt_ptv_flag2 uint8 // flag2:
	//	txt_reserved_1:4        uint8         // reserved
	//	txt_rec_style:4;          // text record style:
	txt_element_esdid uint32
	txt_reserved_2    int32  // reserved for 64 bit...
	txt_offset        int32  // starting
	txt_true_length   int32  // (always zero)
	txt_encoding_type int16  // (always zero)
	txt_data_length   uint16 // total len of txt_data

	// And now the variable part...
	txt_data [TDFIXEDSIZE]byte
}

type goff_continuation_record struct {
	// fixed_part
	cont_ptv_prefix uint8

	cont_ptv_flag uint8 // flag:
	// 	cont_ptv_type:4
	//	cont_ptv_reserved:2;
	//	cont_ptv_continuation:1;   // continuation of previous record
	//	cont_ptv_continued:1;      // continuated on to the next record
	cont_ptv_version uint8 // always 0

	// And now the variable part...
	cont_data [CDFIXEDSIZE]uint8 // data continued...
}

// GOFF RLD record.
// Note: There's a fixed header portion and a variable data portion.
type goff_rld_record struct {
	// rld_fixed_part
	rld_ptv_prefix uint8

	rld_ptv_flag uint8 // flag:
	//  rld_ptv_type:4     uint8
	//  rld_ptv_reserved:2;
	//  rld_ptv_continuation:1;   // continuation of previous record
	//  rld_ptv_continued:1;      // continuated on to the next record
	rld_ptv_version uint8 // always 0
	rld_reserved_1  uint8 // reserved
	rld_data_length int16 // total len of rld_data

	// And now the variable part...
	rld_data [74]uint8
}

// The rld_data area holds 1 or more of these.  An goff_rld_data_item is
// created for each RLI, and appended to the rld_data/rldc_data.
// Note: Some fields apply only for certain RLD types; these are
//       indicated (in parenthesis) in the comments.
// Note: Each goff_rld_data_item can be a different length, due to the
//       presence or absence or different widths of certain fields,
//       as specified by the flag bits.

const (
	// goff_rld_data_item_fixed_part
	// rld_flags               bytes 0..5 mapped below
	// Byte 0
	RLDRT_R_ADDR        = 0
	RLDRT_R_OFFSET      = 1
	RLDRT_R_LENGTH      = 2
	RLDRT_R_VALUE_IMMED = 3
	RLDRT_R_TEXT        = 4
	RLDRT_R_SYMBOL      = 5
	RLDRT_RI_REL        = 6 // long displacement relative
	RLDRT_R_ADA         = 7 // -> XPLink ADA      +++
	RLDRT_R_REL         = 9 // RXY-relative

	// Byte 1
	// rld_R_ptr_indicators
	RLDRO_LABEL   = 0
	RLDRO_ELEMENT = 1
	RLDRO_CLASS   = 2
	RLDRO_PART    = 3

	// Byte 2
	RLDAC_ADD             = 0
	RLDAC_SUB             = 1
	RLDAC_NEG             = 2
	RLDAC_SHIFT           = 3
	RLDAC_MULT            = 4
	RLDAC_DIV_4_QUOTIENT  = 6
	RLDAC_DIV_4_REMAINDER = 7
	RLDAC_AND             = 8
	RLDAC_OR              = 9
	RLDAC_XOR             = 10
	RLDAC_MOVE            = 16
)

type goff_rld_data struct {
	rld_data     *goff_rld_data_item
	rld_type     int32
	rdiRPOff_ptr *goff_rld_data_item_RPOff
	rdiROff_ptr  *goff_rld_data_item_ROff
	rdiR_ptr     *goff_rld_data_item_R
	rdiPOff_ptr  *goff_rld_data_item_POff
	rdiP_ptr     *goff_rld_data_item_P
	rdiRP_ptr    *goff_rld_data_item_RP
	rdinoRP_ptr  *goff_rld_data_item_no_RP
}

type goff_rld_data_item struct {
	// goff_rld_data_item_fixed_part
	// rld_flags               bytes 0..5 mapped below
	// Byte 0
	rld_data_item_byte0 uint8 // byte0:
	//  	rld_same_R_ID:1          // always FALSE
	//	rld_same_P_ID:1;         // always FALSE
	//	rld_same_offset:1;       // always FALSE
	//	rld_reserved_1:1;        // reserved
	//	rld_reserved_2:1;        // reserved
	//	rld_ext_attr_present:1;  // always FALSE
	//	rld_offset_8_bytes:1;    // always FALSE
	//	rld_amode_sensitive:1;   // always FALSE (VCON)
	// Byte 1
	// rld_R_ptr_indicators
	rld_data_item_byte1 uint8 // byte1:
	//	rld_ref_type:4           // reference type:
	//	rld_ref_origin:4;        // reference origin:

	// Byte 2
	rld_data_item_byte2 uint8 // byte2:
	//	rld_action:7    uint8_t  // action or operation:
	//	rld_no_fetch_fixup_field:1;

	// Byte 3
	rld_reserved_3 uint8 // reserved

	// Byte 4
	rld_targ_field_byte_len uint8

	// Byte 5
	rld_data_item_byte5 uint8 // byte5:
	//	rld_bit_length:3
	//	rld_condseq:1;
	//	rld_reserved_4:1;        // reserved
	//	rld_bit_offset:3;

	rld_reserved_5 uint8 // reserved
	rld_reserved_6 uint8 // reserved
}

// this is ugly but must be done to be able to handle
// RLD's containing P_esdid only, R_esdid only, or both
// followed by offset(s) and ext_attr thus we need 3
// overlaid level 3 structures, within a "var_part"
// this may change when we'll support ext_attr
// goff_rld_data_item_var_part
// Note - Essentially this if for compressing the RLD data. So based on
//        the rld_flags, any combination of the R_esdid, P_esdid and
//        the offset may be omitted.
//        For now, we always do goff_rld_data_item_RP.
type goff_rld_data_item_RPOff struct {
	rld_R_esdid int32
	rld_P_esdid int32
	rld_offset4 int32 // we use 4-byte offsets
	// rld_offset8hi rld_offset4         //  (for an
	// rld_offset8lo         int32       //   8-byte offset)
	// rld_ext_attr_id       int32       // (present only if
	// rld_ext_attr_offset;  int32       //   rld_ext_attr_present)
}

type goff_rld_data_item_POff struct {
	rld_P_esdid int32
	rld_offset4 int32 // we use 4-byte offsets
	// rld_offset8hi rld_offset4         // (for an
	// rld_offset8lo         int32       //   8-byte offset)
	// rld_ext_attr_id       int32       // (present only if
	// rld_ext_attr_offset   int32       //   rld_ext_attr_present)
}

type goff_rld_data_item_P struct {
	rld_P_esdid int32
	// rld_offset8hi rld_offset4         // (for an
	// rld_offset8lo          int32      //   8-byte offset)
	// rld_ext_attr_id        int32      // (present only if
	// rld_ext_attr_offset    int32      //   rld_ext_attr_present)
}

type goff_rld_data_item_ROff struct {
	rld_R_esdid int32
	rld_offset4 int32 // we use 4-byte offsets
	// rld_offset8hi rld_offset4         // (for an
	// rld_offset8lo 	int32        //   8-byte offset)
	// rld_ext_attr_id	int32        // (present only if
	// rld_ext_attr_offset	int32        //   rld_ext_attr_present)
}

type goff_rld_data_item_R struct {
	rld_R_esdid int32
	// rld_offset8hi rld_offset4         // (for an
	// rld_offset8lo 	int32        //   8-byte offset)
	// rld_ext_attr_id      int32        // (present only if
	// rld_ext_attr_offset  int32        //   rld_ext_attr_present)
}

type goff_rld_data_item_RP struct {
	rld_R_esdid int32
	rld_P_esdid int32
	// rld_offset8hi rld_offset4          // (for an
	// rld_offset8lo         int32        //   8-byte offset)
	// rld_ext_attr_id       int32        // (present only if
	// rld_ext_attr_offset   int32        //   rld_ext_attr_present)
}

type goff_rld_data_item_no_RP struct {
	rld_offset4 int32 // we use 4-byte offsets
	// rld_offset8hi rld_offset4          // (for an
	// rld_offset8lo      	int32         //   8-byte offset)
	// rld_ext_attr_id      int32         // (present only if
	// rld_ext_attr_offset  int32         //   rld_ext_attr_present)
}

// GOFF END record.
// Note: Since we never use the "requested entry-point provided by
//       external name" form, we never need to continue the END card.
// Note: Because we never use the end_entry_name, our END records are
//       always fixed-length.
const (
	ENDEP_NONE         = 0 //   not requested
	ENDEP_ESDID_OFFSET = 1 //   by esdid and offset

	// ENDEP_EXT_NAME = 2     		//   by ext name (never)
)

type goff_end_record struct {
	// end_fixed_part
	end_ptv_prefix uint8

	end_ptv_flag1 uint8 // flag1:
	//  end_ptv_type:4  uint8
	//  end_ptv_reserved:2;
	//  end_ptv_continuation:1;  // continuation of previous record
	//  end_ptv_continued:1;     // continuated on to the next record

	end_ptv_version uint8 // always 0

	end_ptv_flag2 uint8 // flag2:
	//  end_reserved_1:6 uint8        // reserved
	//  end_ep_request_type:2;   // entry-point requested:

	end_amode        uint8
	end_reserved_2   [3]uint8 // reserved
	end_record_count int32
	end_esdid        int32
	end_reserved_3   int32 // reserved for 64 bit...
	end_offset       int32
	end_name_length  int16 // (always 0)

	// And now the variable part...
	// end_entry_name  [x]int8     	// (never used)
}

//
// (3) Control block declarations follow
//     e.g. CEESTART
//

type celqstrt_text struct {
	// section 2
	nop_1          uint32 // 0x47000000
	nop_2          uint32 // 0x47000002
	stmg           uint32 // 0xEBECD008
	stmg_p2        uint16 // 0x0024
	bru1           uint16 // 0xA7F4000E
	bru2           uint16
	dcsiglen       uint16 // 0x0018
	dcsignature    uint32 // 0xCE03030F
	adparamlist    uint64 // 0x0000000000000038
	signeye        uint64 // 'CEESTART' 0xC3C5C5E2E3C1D9E3
	xplmainreserve uint16 // 0x0100

	// section 3
	balr  uint16 // 0x0530
	lg    uint32 // 0xE3F03064
	lg_p2 uint16 // 0x0004
	balr2 uint16 // 0x050F

	// section 4
	reserve2    uint32
	adcelqmain  uint64
	versionmark uint16 // 0xFFFD (-3)
	al2stlen    uint16 // 0x0058
	padding     uint32 // 0x0000
	reserve3    [3]uint64
	adsignature uint64 // 0x0000000000000012
	reserve4    uint64
	adcelqfman  uint64
	adcelqllst  uint64
	reserve5    uint64
	adcelqetbl  uint64

	// section 5
	adcelqbst uint64
}

type celqmain_text_xplink struct {
	first_word uint64 // 0x0400000100000000 (rent) or 0x0500000100000000 (norent)
	admain     uint64 // AD(main)
	adcelqinpl uint64 // AD(CELQINPL)
	adenv      uint64 // AD(environment) - for NORENT
	//   a0  	uint32  // for RENT
	//   qenv  	uint32  // Q(environment) - for RENT
}

type PPA1 struct {
	_version               uint8
	_LEsignature           uint8
	_savedGPRmask          uint16
	_ppa2offset            uint32
	_flags1                uint8
	_flags2                uint8
	_flags3                uint8
	_flags4                uint8
	_parmslength           uint16
	_prologlength          uint8
	_allocaregAndchgoffset uint8
	_codelength            uint32
	_funcnamelength        uint16
	//_entrypointname       []byte
}

type commonPPA2 struct {
	_memberIdAndSubId uint16 // language dependent
	_memberdefined    uint8  // c370_plist+c370_env
	_controllevel     uint8  // X'04' for xplink otherwise X'03'
	_ceestartOffset   int32  // signed offset to start of CEESTART
	_cdiOffset        int32  // signed offset to CDI or 0 if nodebug - i.e. PPA4
	_tsOffset         int32  // signed offset to timestamp
	_epOffset         int32  // signed offset to primary entrypoint (always 0)

	// Compilation flags
	_ppa2_flag uint32 // flag
	//  _fw_ieee:1;         // set if compiled with float(ieee)
	//  _fw_libcomp:1;      // set if this is associated with library code
	//  _fw_srvlevel:1;     // set if program contains service information
	//  _fw_storeargs:1;    // set if xplink(storeargs)

	//  _fw_reserved1:1;    //
	//  _fw_charset_bias:1; // set if compiled with ASCII
	//  _fw_srvcomnt:1;     // set if additional service comments - SOS or PLI service
	//  _fw_xplink:1;       // set if compiled with xplink

	//  _fw_reserved2:1;    //
	//  _fw_md5:1;          // set if MD5 located 16 bytes before timestamp
	//  _fw_afpvol:1;       // set if all AFP registers volatile
	//  _fw_reserved3:21;
}

type dwarf64BitPPA4 struct { // format: same dwarf64BitPPA4
	_flags1 uint32
	_flags2 uint32

	_ROStaticAddress   uint64
	_RWStaticOffset    uint64
	_symbolTableOffset uint64
	_codeOffset        uint64 // A(code-PPA4)
	_codeSize          uint64
	_dwarfImageOffset  uint64 // A(dwarfImage-PPA4) or reserved

	// following variable part follows
	// 4 byte name length followed by Dwarf  side filename
	// 2 byte name length followed by Source filename
	_varBegin uint16 // where variable stuff starts
}

const (
	ObjectCode32BitRelativeRelocation = 0 // p <- A(r1-r2)
	ObjectCodeRelativeRelocation      = 1 // p <- A(r1-r2)    32/64 size depending on codegen
	ObjectCodeAddressRelocation       = 2 // p <- A(r1)       32/64 size depending on codegen
	ObjectCode32BitAddressRelocation  = 3 // p <- A(r1)       32 bit size
	ObjectCode64BitAddressRelocation  = 4 // p <- A(r1)       64 bit size
	ObjectCodeSectionOffsetRelocation = 5 // p <- SectionOffset(r1)  32/64 size depending on codegen (no linker reloc here)
	ObjectCode32BitQCon               = 6 // p <- Q(r1)
	ObjectCodeSizeofCodeRelocation    = 7 // p <- sizeof(code setion) 32/64 size depending on codegen
	// chwan - this was never used before.
	//         It is now used to represent the XPLINK function descriptor
	ObjectCodeADARelocation = 8 // p <- A(ADA(r1))  32/64 size depending on codegen
)

type objectCodeSnippet struct {
	_snippetName          string
	_snippetSectionOffset uint64 // object file ESD offset
	_snippetTxi           uint32
}

// Relocation for object code
type objectCodeRelocation struct {
	_type  uint8
	_pptr  *LSym
	_poff  int32
	_r1ptr *LSym
	_r1off int32
}

var ghdr goff_hdr_record
var gend goff_end_record
var Isgoff bool
var _amode uint8
var _rmode uint8
var _next_esdid uint32
var _next_exi_ix uint32
var _cummulativeTXTSize uint64
var _celqstrt_offset uint64
var _next_txi_ix uint32
var _next_rli_ix uint32
var _ccsect_exi_ix uint32
var _code_exi_ix uint32
var _ccsect_rld_exi_ix uint32
var _static_exi_ix uint32
var _celqstrt_exi_ix uint32
var _start_code_exi_ix uint32
var _start_er_exi_ix uint32
var _exi []exi_entry
var _txi []txi_entry
var _rli []rli_entry
var _ppa2Offset uint64
var _ppa4Ptr *dwarf64BitPPA4
var _ppa2Buffer commonPPA2
var _ppa4 dwarf64BitPPA4
var _ppa1SnippetList []*objectCodeSnippet
var _objectCodeRelocationList []*objectCodeRelocation
var _shash map[int64]uint32

// Dwarf Debug
type dwarf_section struct {
	name   string
	refESD uint32
	defESD uint32
	sec    *LSym
	next   *dwarf_section
}

var _debugSectionList *dwarf_section
var _debugSectionListTail *dwarf_section
var _debug_abbrev dwarf_section
var _debug_line dwarf_section
var _debug_frame dwarf_section
var _debug_info dwarf_section
var _debug_pubnames dwarf_section
var _debug_pubtypes dwarf_section
var _debug_aranges dwarf_section

const (
	maxTotalTXTSize  = 1 << 30
	maxTotalExi      = 200000
	maxTotalTxi      = 500000
	maxTotalRli      = 600000
	TDFIXEDSIZE      = 56
	CDFIXEDSIZE      = 77
	MAX_VAR_DATA_LEN = 16 * 1024
	LE_VV_RR         = 0x20F
	SIZE_ADDR_CON_64 = 8
	SIZE_ADDR_CON_32 = 4
)

const (
	NONE        = 0x000
	SAME_PID    = 0x001
	SAME_RID    = 0x010
	SAME_OFFSET = 0x100
)

/*
 Initialize the global variable that describes the GOFF header. It will be updated as
 we write section and prog headers.
*/
func Goffinit() {
	Isgoff = true

	_amode = AMODE_64
	_rmode = RMODE_64
	_next_exi_ix = 0
	_next_esdid = 0
	_next_txi_ix = 0
	_cummulativeTXTSize = 0
	_next_rli_ix = 0
	_static_exi_ix = 0
	_ccsect_exi_ix = 0
	_code_exi_ix = 0
	_start_er_exi_ix = 0
	_exi = make([]exi_entry, maxTotalExi)
	_txi = make([]txi_entry, maxTotalTxi)
	_rli = make([]rli_entry, maxTotalRli)
	ppa2Init()
	if Debug['w'] == 0 { // dwarf enable
		dwarfSectionInit()
	}
}

func dwarfSectionInit() {
	_debug_abbrev.name = "D_ABREV"
	_debugSectionList = &_debug_abbrev

	_debug_line.name = "D_LINE"
	_debug_line.sec = linesec
	_debugSectionList.next = &_debug_line
	_debugSectionListTail = &_debug_line

	_debug_frame.name = "D_FRAME"
	_debug_frame.sec = framesec
	_debugSectionListTail.next = &_debug_frame
	_debugSectionListTail = &_debug_frame

	_debug_info.name = "D_INFO"
	_debug_info.sec = infosec
	_debugSectionListTail.next = &_debug_info
	_debugSectionListTail = &_debug_info

	_debug_pubnames.name = "D_PBNMS"
	_debugSectionListTail.next = &_debug_pubnames
	_debugSectionListTail = &_debug_pubnames

	_debug_pubtypes.name = "D_PTYPES"
	_debugSectionListTail.next = &_debug_pubtypes
	_debugSectionListTail = &_debug_pubtypes

	_debug_aranges.name = "D_ARNGE"
	_debug_aranges.sec = arangessec
	_debugSectionListTail.next = &_debug_aranges
	_debugSectionListTail = &_debug_aranges

}

func ppa2Init() {

	_ppa2Buffer._memberIdAndSubId = 0x0360 // use the member ID for C(x'03'), along with a sub id of x'60' for GO
	_ppa2Buffer._memberdefined = 0x22      // TODO: need c370_plist+c370_env
	_ppa2Buffer._controllevel = 0x04

	// A(CEESTART-PPA2)
	_ppa2Buffer._ceestartOffset = 0

	// A(PPA2-PPA4)
	_ppa2Buffer._cdiOffset = 0

	// A(TIMESTAMP-PPA2): create relocation for later processing
	_ppa2Buffer._tsOffset = 0

	// Set flags:
	//   _fw_ieee = true
	//   _fw_storargs = true
	//   _fw_charset_bias = true (ASCII)
	//   _fw_xplink = true
	_ppa2Buffer._ppa2_flag = 0x95000000
}

// encodeSym converts the given symbol name (UTF-8) into EBCDIC-037.
func encodeSym(name string) []byte {

	enc, err := EncodeStringEBCDIC(name)
	if err != nil {
		// Unable to encode the given symbol in EBCDIC. Replace with a sha1 hash for now.
		hash := sha1.Sum([]byte(name))
		enc, err = EncodeStringEBCDIC(fmt.Sprintf("goffsym_%x", hash[:8])) // use only the first 8 bytes
		if err != nil {
			Diag("could not convert '%v' to EBCDIC", name)
		}
	}
	return enc
}

func getNextExiIx() uint32 {
	_next_exi_ix++
	return _next_exi_ix
}

func getNewExiEntry(ix uint32) *exi_entry {
	if ix >= maxTotalExi {
		exiEntry := new(exi_entry)
		_exi = append(_exi, *exiEntry)
	}

	return &_exi[ix]
}

func getNextTxiIx() uint32 {
	_next_txi_ix++
	return _next_txi_ix
}

func getNewTxiEntry(ix uint32) *txi_entry {
	if ix >= maxTotalTxi {
		txiEntry := new(txi_entry)
		_txi = append(_txi, *txiEntry)
	}
	return &_txi[ix]
}

func getNextRliIx() uint32 {
	_next_rli_ix++
	return _next_rli_ix
}

func getNewRliEntry(ix uint32) *rli_entry {
	if ix >= maxTotalRli {
		rliEntry := new(rli_entry)
		_rli = append(_rli, *rliEntry)
	}
	return &_rli[ix]
}

func getNextEsdId() uint32 {
	_next_esdid++
	return _next_esdid
}

func Asmbgoffsetup() {

	_shash = make(map[int64]uint32)

	// CODE: SD
	sd := getNewExiEntry(_ccsect_exi_ix)
	addDefExiCode(sd, "GO#C", 0, EXT_SD)
	sd.exf_xplink = false
	sd.exf_force_rent = true

	// CODE: ED child
	_code_exi_ix = getNextExiIx()
	ed := getNewExiEntry(_code_exi_ix)
	addDefExiCode(ed, "G_CODE64", _ccsect_exi_ix, EXT_ED)
	ed.exi_alignment = EXIAL_DWORD
	ed.exi_length = 0 // length value filled in later in buildCodePart()

	// CODE: LD child
	_ccsect_rld_exi_ix = getNextExiIx()
	ld := getNewExiEntry(_ccsect_rld_exi_ix)
	addDefExiCode(ld, "GO#C", _code_exi_ix, EXT_LD)
}

func getGoffhdr() *goff_hdr_record {
	return &ghdr
}

func getGoffend() *goff_end_record {
	return &gend
}

func setBindingScope(ee exi_entry) uint8 {
	if ee.exf_export || (ee.exf_xplink && !ee.exf_def) {
		return ESDSC_EXPORT_IMPORT
	} else if ee.exf_indirect && !ee.exf_internal {
		return ESDSC_MODULE
	} else if ee.exf_internal {
		return ESDSC_SECTION
	} else {
		return ESDSC_LIBRARY
	}
}

func getDataExi(symbolName string, size int64) uint32 {
	var ee *exi_entry
	isRentSymbol := false
	alignment := 0

	// PR/LD child: exi_ix here used in relocations
	data_exi_ix := getNextExiIx()
	ee = getNewExiEntry(data_exi_ix)
	if isRentSymbol {
		addDefExiData(ee, symbolName, _code_exi_ix, EXT_PR)
	} else {
		addDefExiCode(ee, symbolName, _code_exi_ix, EXT_LD)
		ee.exf_xplink = false
		ee.exi_offset = uint32(_exi[_code_exi_ix].exi_length)
	}

	if isRentSymbol {
		ee.exi_length = uint32(size)
	}

	ee.exf_mapped = false
	ee.exf_def = true
	ee.exf_deferred = isRentSymbol
	ee.exf_export = false
	ee.exf_executable = false
	ee.exi_alignment = uint32(alignment)

	return data_exi_ix
}

func getDataExiForLSym(sym *LSym) uint32 {

	isRentSymbol := false
	symbolName := sym.Name
	alignment := 4

	// PR/LD child: exi_ix here used in relocations
	data_exi_ix := getNextExiIx()
	ee := getNewExiEntry(data_exi_ix)
	if _ccsect_rld_exi_ix == 0 {
		_ccsect_rld_exi_ix = data_exi_ix
	}

	if isRentSymbol {
		addDefExiData(ee, symbolName, _code_exi_ix, EXT_PR)
	} else {
		addDefExiCode(ee, symbolName, _code_exi_ix, EXT_LD)
		hashing(sym, data_exi_ix)
		ee.exf_xplink = true
		ee.exi_offset = uint32(_exi[_code_exi_ix].exi_length)
	}

	if isRentSymbol {
		ee.exi_length = uint32(sym.Size)
	}

	ee.exf_mapped = false
	ee.exf_def = true
	ee.exf_deferred = isRentSymbol
	ee.exf_export = false // TODO
	ee.exf_executable = false
	ee.exi_alignment = uint32(alignment)

	return data_exi_ix
}

func hashing(s *LSym, data_ld_exi_ix uint32) {
	j := _shash[s.Value]
	if j == 0 {
		_shash[s.Value] = data_ld_exi_ix
	}
}

func createObjCodeRelocation(s *LSym) {
	var r *Reloc

	for i := int64(0); i < int64(len(s.R)); i++ {
		r = &s.R[i]
		// chwan - I hope it is ok for not checking if goos = "zos" as goff.go is for zos only.
		// Identify if the referenced is a user C function.
		// Or the referenced is one of the cgo runtime C functions
		// If so, we want to emit the 16-byte relocatable that has two parts.
		if strings.HasPrefix(r.Sym.Name, "_cgo_") && strings.Contains(r.Sym.Name, "_Cfunc_") && r.Siz == 16 {
			objReloc := objectCodeRelocation{ObjectCodeADARelocation, s, r.Off, r.Sym, int32(r.Add)}
			_objectCodeRelocationList = append(_objectCodeRelocationList, &objReloc)
		} else if (r.Sym.Name == "x_cgo_thread_start" ||
			r.Sym.Name == "x_cgo_sys_thread_create" ||
			r.Sym.Name == "x_cgo_notify_runtime_init_done" ||
			r.Sym.Name == "x_cgo_init" ||
			r.Sym.Name == "x_cgo_malloc" ||
			r.Sym.Name == "x_cgo_free") && r.Siz == 16 {
			objReloc := objectCodeRelocation{ObjectCodeADARelocation, s, r.Off, r.Sym, int32(r.Add)}
			_objectCodeRelocationList = append(_objectCodeRelocationList, &objReloc)
		} else {
			objReloc := objectCodeRelocation{ObjectCodeAddressRelocation, s, r.Off, r.Sym, int32(r.Add)}

			_objectCodeRelocationList = append(_objectCodeRelocationList, &objReloc)
		}
	}
}

func addBssSectionTxt(sname string, s *LSym, addr int64, size int64) {
	var pad uint32

	eaddr := addr + size
	z := byte(0)

	if Debug['a'] != 0 {
		fmt.Fprintf(&Bso, "Add bss section %s \n", sname)
	}

	for ; s != nil; s = s.Next {
		if s.Type&obj.SSUB == 0 && s.Value >= addr {
			break
		}
	}

	var sectsym *LSym
	var esectsym *LSym

	switch sname {
	case ".noptrbss":
		sectsym = Linklookup(Ctxt, "runtime.noptrbss", 0)
		esectsym = Linklookup(Ctxt, "runtime.enoptrbss", 0)
	case ".bss":
		sectsym = Linklookup(Ctxt, "runtime.bss", 0)
		esectsym = Linklookup(Ctxt, "runtime.ebss", 0)
	case ".tbss":
		sectsym = Linklookup(Ctxt, "runtime.tbss", 0)
		esectsym = Linklookup(Ctxt, "runtime.etbss", 0)
	}

	data_exi_ix := getDataExi(sectsym.Name, 0)
	j := _shash[sectsym.Value]
	if j == 0 {
		_shash[sectsym.Value] = data_exi_ix
	}

	for ; s != nil; s = s.Next {
		if s.Type&obj.SSUB != 0 {
			continue
		}
		if s.Value >= eaddr {
			break
		}

		Ctxt.Cursym = s
		totalSize := int64(0)
		var buffer []byte

		for i := int64(0); i < s.Size; i++ {
			buffer = append(buffer, z)
			totalSize++
			addr++
		}

		if s.Next != nil {
			pad = 0
			for ; addr < s.Next.Value; addr++ {
				buffer = append(buffer, z)
				totalSize++
				pad++
			}
		}

		if totalSize > 0 {
			data_exi_ix := getDataExi(s.Name, totalSize)
			addTxi(_code_exi_ix, 0, int64(len(buffer)), &buffer, 0)
			j := _shash[s.Value]
			if j == 0 {
				_shash[s.Value] = data_exi_ix
			}
		}

		if Debug['a'] != 0 {
			q := s.P
			addr := s.Value - INITTEXT + 0x6e
			fmt.Fprintf(&Bso, "%.6x\t%-20s len = 0x%x size = 0x%x\n", uint64(int64(s.Value)), s.Name, len(q), s.Size)
			fmt.Fprintf(&Bso, "\t%.6x\t  len = 0x%x\n", addr, totalSize)
		}
	}

	data_exi_ix = getDataExi(esectsym.Name, 0)
	_exi[data_exi_ix].exi_offset = uint32(_exi[_code_exi_ix].exi_length - pad)
	j = _shash[esectsym.Value]
	if j == 0 {
		_shash[esectsym.Value] = data_exi_ix
	}
}

func addDataSectionTxt(sname string, s *LSym, addr int64, size int64) {
	eaddr := addr + size
	var p []byte
	var ep []byte
	var z byte
	var pad uint32
	var data_exi_ix uint32
	var j uint32

	if Debug['a'] != 0 {
		fmt.Fprintf(&Bso, "Add data section %s \n", sname)
	}

	for ; s != nil; s = s.Next {
		if s.Type&obj.SSUB == 0 && s.Value >= addr {
			break
		}
	}

	var sectsym *LSym
	var esectsym *LSym

	switch sname {
	case ".rodata":
		sectsym = Linklookup(Ctxt, "runtime.rodata", 0)
		esectsym = Linklookup(Ctxt, "runtime.erodata", 0)
	case ".typelink":
		sectsym = Linklookup(Ctxt, "runtime.typelink", 0)
		esectsym = Linklookup(Ctxt, "runtime.etypelink", 0)
	case ".gosymtab":
		sectsym = Linklookup(Ctxt, "runtime.gosymtab", 0)
		esectsym = Linklookup(Ctxt, "runtime.egosymtab", 0)
	case ".gopclntab":
		sectsym = Linklookup(Ctxt, "runtime.gopclntab", 0)
		esectsym = Linklookup(Ctxt, "runtime.egopclntab", 0)
	case ".shstrtab":
		sectsym = Linklookup(Ctxt, "runtime.shstrtab", 0)
		esectsym = Linklookup(Ctxt, "runtime.eshstrtab", 0)
	case ".noptrdata":
		sectsym = Linklookup(Ctxt, "runtime.noptrdata", 0)
		esectsym = Linklookup(Ctxt, "runtime.enoptrdata", 0)
	case ".data":
		sectsym = Linklookup(Ctxt, "runtime.data", 0)
		esectsym = Linklookup(Ctxt, "runtime.edata", 0)
	}

	if sectsym != nil {
		data_exi_ix = getDataExi(sectsym.Name, 0)
		j = _shash[sectsym.Value]
		if j == 0 {
			_shash[sectsym.Value] = data_exi_ix
		}
	}

	for ; s != nil; s = s.Next {
		if s.Type&obj.SSUB != 0 {
			continue
		}
		if s.Value >= eaddr {
			break
		}

		Ctxt.Cursym = s
		var buffer []byte

		if s.Value < addr {
			Diag("phase error: addr=%#x but sym=%#x type=%d", int64(addr), int64(s.Value), s.Type)
			errorexit()
		}

		p = s.P
		ep = p[len(s.P):]
		for -cap(p) < -cap(ep) {
			buffer = append(buffer, uint8(p[0]))
			p = p[1:]
		}

		addr += int64(len(s.P))
		for ; addr < s.Value+s.Size; addr++ {
			buffer = append(buffer, z)
		}

		if s.Next != nil {
			pad = 0
			for ; addr < s.Next.Value; addr++ {
				buffer = append(buffer, z)
				pad++
			}
		}

		if Debug['a'] != 0 {
			dump(s)
		}

		totalSize := int64(len(buffer))
		if totalSize > 0 {
			data_exi_ix := getDataExi(s.Name, totalSize)
			addTxi(_code_exi_ix, 0, totalSize, &buffer, 0)
			j := _shash[s.Value]
			if j == 0 {
				_shash[s.Value] = data_exi_ix
			}
		}

		if int64(len(s.R)) > 0 {
			createObjCodeRelocation(s)
		}
	}

	if esectsym != nil {
		data_exi_ix = getDataExi(esectsym.Name, 0)
		_exi[data_exi_ix].exi_offset = uint32(_exi[_code_exi_ix].exi_length - pad)
		j = _shash[esectsym.Value]
		if j == 0 {
			_shash[esectsym.Value] = data_exi_ix
		}
	}
}

func dump(s *LSym) {
	q := s.P
	addr := s.Value

	fmt.Fprintf(&Bso, "%.6x\t%-20s len = 0x%x size = 0x%x\n", uint64(int64(s.Value)), s.Name, len(q), s.Size)

	for len(q) >= 16 {
		fmt.Fprintf(&Bso, "%.6x\t% x\n", uint64(addr), q[:16])
		addr += 16
		q = q[16:]
	}

	if len(q) > 0 {
		fmt.Fprintf(&Bso, "%.6x\t% x\n", uint64(addr), q)
		addr += int64(len(q))
	}

	var r *Reloc
	var rsname string
	var typ string

	for i := int64(0); i < int64(len(s.R)); i++ {
		r = &s.R[i]
		rsname = ""
		if r.Sym != nil {
			rsname = r.Sym.Name
		}
		typ = "?"
		switch r.Type {
		case obj.R_ADDR:
			typ = "addr"

		case obj.R_PCREL:
			typ = "pcrel"

		case obj.R_CALL:
			typ = "call"
		}

		fmt.Fprintf(&Bso, "\treloc %.8x/%d %s %s+%#x [%#x]\n", uint(s.Value+int64(r.Off)), r.Siz, typ, rsname, int64(r.Add), int64(r.Sym.Value+r.Add))
	}
}

func buildCODEPart() {

	var data_exi_ix uint32
	var isXPLinkEntry bool

	// start at INITTEXT like LoZ
	elems := uint64(INITTEXT) - uint64(_exi[_code_exi_ix].exi_length)
	if elems > 0 {
		pad := make([]byte, elems)
		addTxi(_code_exi_ix, 0, int64(elems), &pad, 0)
	}

	sectname := "runtime.text"
	esectname := "runtime.etext"
	getDataExi(sectname, 0)

	// Emit the TXT for the code - emit required ESDs for entry points
	for s := Ctxt.Textp; s != nil; s = s.Next {

		if s.Type&obj.SSUB != 0 {
			continue
		}

		isXPLinkEntry = false
		var i int

		for i = range _ppa1SnippetList {
			if _ppa1SnippetList[i]._snippetName == s.Name {
				isXPLinkEntry = true
				break
			}
		}

		// Write the epm_to_ppa1_offset to EPM
		if isXPLinkEntry {
			var ppa1_location uint64

			ppa1_location = _ppa1SnippetList[i]._snippetSectionOffset
			epm_to_ppa1_offset := int32(ppa1_location - _cummulativeTXTSize)
			//        fmt.Printf("epm_to_ppa1_offset = %08x\n", epm_to_ppa1_offset)
			var epm_buf bytes.Buffer
			binary.Write(&epm_buf, binary.BigEndian, &epm_to_ppa1_offset)
			mem := *(*[]byte)(unsafe.Pointer(&epm_buf))
			const EPM_PPA1_OFFSET = 8

			(s.P)[EPM_PPA1_OFFSET] = mem[0]
			(s.P)[EPM_PPA1_OFFSET+1] = mem[1]
			(s.P)[EPM_PPA1_OFFSET+2] = mem[2]
			(s.P)[EPM_PPA1_OFFSET+3] = mem[3]

			// TODO: LEinitSize is defined in rt0_zos_s390x.s, make it common
			epm_buf.Reset()
			epm_dsasize := int32(0x00010404) // LEinitSize/32 + alloca flag on
			binary.Write(&epm_buf, binary.BigEndian, &epm_dsasize)
			mem = *(*[]byte)(unsafe.Pointer(&epm_buf))
			const EPM_DSASIZE_OFFSET = 12

			(s.P)[EPM_DSASIZE_OFFSET] = mem[0]
			(s.P)[EPM_DSASIZE_OFFSET+1] = mem[1]
			(s.P)[EPM_DSASIZE_OFFSET+2] = mem[2]
			(s.P)[EPM_DSASIZE_OFFSET+3] = mem[3]

			/*
			   fmt.Printf("s.P ")
			   for i:=0; i<16; i++ {
			           fmt.Printf("%02x ",(s.P)[i])
			   }
			   fmt.Printf("\n");
			*/

		}

		/*
		   fmt.Printf("address: %08x  ", _cummulativeTXTSize)
		   fmt.Printf("size: %08x   ", s.Size)
		   fmt.Printf("function: %s\n", s.Name)
		*/

		data_exi_ix = getDataExiForLSym(s)

		if Debug['a'] != 0 {
			dump(s)
		}

		// skip the EPM for LD
		if isXPLinkEntry {
			_exi[data_exi_ix].exi_offset += 16
		}

		addTxi(_code_exi_ix, 0, s.Size, &s.P, 0)
	}

	data_ix := getDataExi(esectname, 0)
	_exi[data_ix].exi_offset = uint32(_exi[_code_exi_ix].exi_length)

	// Emit the TXT for the constant area
	sect := Segtext.Sect.Next
	va := uint64(sect.Vaddr)
	n := '1'

	end := uint64(INITTEXT) + Segtext.Sect.Length
	elems = uint64(0)

	if sect.Vaddr > 0 {
		elems = sect.Vaddr - uint64(end)
	}

	if elems > 0 {
		pad := make([]byte, elems)
		addTxi(_code_exi_ix, 0, int64(elems), &pad, 0)
		va += uint64(elems)
	}

	for ; sect != nil; sect = sect.Next {
		if sect.Length == 0 && sect.Next != nil {
			elems = sect.Next.Vaddr - sect.Vaddr
			if elems > 0 {
				pad := make([]byte, elems)
				addTxi(_code_exi_ix, 0, int64(elems), &pad, 0)
				va += uint64(elems)
			}
		}

		addDataSectionTxt(sect.Name, datap, int64(sect.Vaddr), int64(sect.Length))
		n++
		va = sect.Vaddr + sect.Length
	}

	// Emit the TXT for the data area
	sect = Segdata.Sect

	end = uint64(0)

	for ; sect != nil; sect = sect.Next {

		if strings.Compare(sect.Name, ".noptrdata") == 0 ||
			strings.Compare(sect.Name, ".data") == 0 {
			va = uint64(sect.Vaddr)
			addDataSectionTxt(sect.Name, datap, int64(sect.Vaddr), int64(sect.Length))
			end = sect.Vaddr + sect.Length
			va = sect.Vaddr + sect.Length
			n++
		} else {
			addBssSectionTxt(sect.Name, datap, int64(sect.Vaddr), int64(sect.Length))
			end = sect.Vaddr + sect.Length
			va += sect.Length
			n++
		}
	}
}

func alignaddress(address uint64, alignment uint64) uint32 {
	//	fmt.Printf("original address: %08x\t", address)
	var newaddress uint64
	var padding uint64
	if address%(2<<alignment) != 0 {
		newaddress = (address>>alignment + 1) << alignment
		padding = newaddress - address
	}
	//	fmt.Printf("aligned address: %08x\t padding: %08x\n", newaddress, padding)
	return uint32(padding)

}

func buildPPA1() {
	// ED: PPA1
	var ppa1_exi_ix uint32
	ppa1_exi_ix = _code_exi_ix

	// Add LD for PPA1
	ppa1_ld_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ppa1_ld_exi_ix)
	addDefExiCode(ee, "PPA1", ppa1_exi_ix, EXT_LD)
	ee.exf_xplink = false
	ee.exi_offset = uint32(_exi[ppa1_exi_ix].exi_length)

	var s1 *LSym
	var ppa1 PPA1

	for i := range s390x.XPLinkFunc {
		s1 = Linklookup(Ctxt, s390x.XPLinkFunc[i], 0)
		if s1 == nil {
			Diag("No entry point is found!")
		}
		ppa1_p := createPPA1(s1)
		paddinglen := alignaddress(uint64(unsafe.Sizeof(ppa1))-2+uint64(ppa1_p._funcnamelength), EXIAL_FWORD)

		ppa1Snippet := new(objectCodeSnippet)
		_ppa1SnippetList = append(_ppa1SnippetList, ppa1Snippet)

		ppa1Snippet._snippetName = s390x.XPLinkFunc[i]
		ppa1Snippet._snippetSectionOffset = _cummulativeTXTSize

		createPPA1Txt(s1, ppa1_p, paddinglen)
		ppa1Snippet._snippetTxi = _exi[_code_exi_ix].exi_last_txi_ix
	}
}

func createPPA1Txt(s *LSym, ppa1 *PPA1, paddinglen uint32) {
	var ppa1_mem []byte
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, ppa1)
	mem := *(*[]byte)(unsafe.Pointer(&bin_buf))
	ppa1_mem = append(ppa1_mem, mem...)
	ppa1_mem = append(ppa1_mem, encodeSym(s.Name)...)
	//        fmt.Printf("len of ppa1_mem is %08x\n", len(ppa1_mem))
	if paddinglen > 0 {
		padding := make([]byte, paddinglen)
		ppa1_mem = append(ppa1_mem, padding...)
	}

	//	fmt.Printf("len of ppa1_mem is %08x\n", len(ppa1_mem))
	addTxi(_code_exi_ix, 0, int64(len(ppa1_mem)), &ppa1_mem, TXTTS_BYTE)
}

func createPPA1(s *LSym) *PPA1 {
	var _ppa1Buffer PPA1
	_ppa1Buffer._version = 0x02
	_ppa1Buffer._LEsignature = 0xCE
	_ppa1Buffer._savedGPRmask = 0x0FFF
	_ppa1Buffer._ppa2offset = 0x00000000
	_ppa1Buffer._flags1 = 0x80
	_ppa1Buffer._flags2 = 0x00
	_ppa1Buffer._flags3 = 0x00
	_ppa1Buffer._flags4 = 0x01

	_ppa1Buffer._parmslength = uint16(s.Args / 4)

	_ppa1Buffer._prologlength = 0x09
	_ppa1Buffer._allocaregAndchgoffset = 0x06
	_ppa1Buffer._codelength = uint32(s.Size)
	_ppa1Buffer._funcnamelength = uint16(len(s.Name))
	return &_ppa1Buffer
}

func buildPPA2() {

	var data_exi_ix uint32

	data_exi_ix = _code_exi_ix

	// A(PPA2-PPA4)
	// PPA4 is right after PPA2
	if Debug['w'] == 0 { // dwarf enable
		_ppa2Buffer._cdiOffset = int32(unsafe.Sizeof(_ppa2Buffer))
	}

	// Add RLD for CELQSTRT offset in PPA2
	ppa2_ld_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ppa2_ld_exi_ix)
	addDefExiCode(ee, "PPA2", _code_exi_ix, EXT_LD)
	ee.exf_xplink = false
	ee.exi_offset = uint32(_exi[data_exi_ix].exi_length)

	goc_ref_exi_ix := getNextExiIx()
	ee = getNewExiEntry(goc_ref_exi_ix)
	addRefExiData(ee, "GO#C", _ccsect_exi_ix, EXT_ER)
	ee.exf_xplink = true
	ee.exf_weak_ref = true

	celqstrt_ref_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqstrt_ref_exi_ix)
	addRefExiData(ee, "CELQSTRT", _ccsect_exi_ix, EXT_ER)
	ee.exf_xplink = false
	ee.exf_weak_ref = true

	// Add relocation to resolve offset from PPA2 to celqstrt: celqstrt - goc - ppa2_offset
	offset := int32(_exi[data_exi_ix].exi_length + uint32(unsafe.Offsetof(_ppa2Buffer._ceestartOffset)))
	addRli(celqstrt_ref_exi_ix, data_exi_ix, SIZE_ADDR_CON_32, RS_POS, offset)
	addRli(goc_ref_exi_ix, data_exi_ix, SIZE_ADDR_CON_32, RS_NEG, offset)

	_ppa2Buffer._ceestartOffset = int32(-_exi[data_exi_ix].exi_length)
	//fmt.Printf("_ppa2Buffer._ceestartOffset = %08x\n", _ppa2Buffer._ceestartOffset)

	// Add TXI for PPA2 (compile unit metadata)
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, &_ppa2Buffer)
	ppa2_mem := *(*[]byte)(unsafe.Pointer(&bin_buf))
	/*
	   for i:=0; i<12; i++ {
	           fmt.Printf("%02x  ", ppa2_mem[i])
	   }
	   fmt.Printf("\n")
	*/

	// update ppa2offset in PPA1
	for i := range _ppa1SnippetList {
		data := *_txi[_ppa1SnippetList[i]._snippetTxi].txi_data_ptr

		ppa2_to_ppa1_offset := int32(_cummulativeTXTSize - _ppa1SnippetList[i]._snippetSectionOffset)
		var ppa2_buf bytes.Buffer
		binary.Write(&ppa2_buf, binary.BigEndian, &ppa2_to_ppa1_offset)
		mem := *(*[]byte)(unsafe.Pointer(&ppa2_buf))
		const PPA2_PPA1_OFFSET = 4
		(data)[PPA2_PPA1_OFFSET] = mem[0]
		(data)[PPA2_PPA1_OFFSET+1] = mem[1]
		(data)[PPA2_PPA1_OFFSET+2] = mem[2]
		(data)[PPA2_PPA1_OFFSET+3] = mem[3]
	}
	addTxi(data_exi_ix, 0, int64(len(ppa2_mem)), &ppa2_mem, TXTTS_BYTE)
}

func buildPPA4() {

	_ppa4Ptr = &_ppa4
	_ppa4._flags1 = uint32(0)

	//
	// PART 1: calculate lengths of entire PPA4 block (and some other values)
	//
	// ppaFixedSize := int64 (unsafe.Offsetof(_ppa4._varBegin))

	// Calculate length of file name and cu name information

	//
	// PART 2: fill in the data
	//   Note some fields like flags and QCON to WSA are at
	//   same offsets in either version 1 or version 2 PPA4
	//
	_ppa4._flags1 |= 0x20000000 // DWARF enbedded in D_x GOFF NOLOAD classes

	// Flags2
	_ppa4._flags2 = uint32(0)

	// PPA4 program flags
	// PPA4 program flags - PPA4 offset X'04' are shown in the following code example:
	// '00000000 00000... ........ ........'B Reserved
	// '........ .....0.. ........ ........'B 31-bit compile
	// '........ .....1.. ........ ........'B 64-bit compile
	// '........ ......00 ........ ........'B Reserved
	// '........ ........ xxxxxxxx ........'B PPA4 version
	//      0: DWARF information not present
	//      1: COBOL V5 and GO PPA4
	//      2: C/C++ DEBUG(FORMAT(DWARF)) PPA4
	// '........ ........ ........ xxxxxxxx'B Offset to file name (zero if not applicable)
	//      file name is prefixed with 4 bytes string length
	//      PPA4 version is 0: unsigned offset from PPA4 to source file name
	//      PPA4 version is 2: unsigned offset from PPA4 to DWARF sidefile name

	_ppa4._flags2 |= 0x00040000 // Set mode32/64 bit (13)

	// bits [16:23] version number: 0x01 for GO
	_ppa4._flags2 |= 0x00000100

	// TO DO
	// Any static?
	// RO Static
	// RW static

	// (version 1) DATA24 offset  or (version 2) SOT address
	// Code size
	// Emit source file table

	var ppa4_exi_ix uint32
	ppa4_exi_ix = _code_exi_ix

	// Add RLD for Code offset: A(code-PPA4)
	ppa4_ld_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ppa4_ld_exi_ix)
	addDefExiCode(ee, "PPA4", _code_exi_ix, EXT_LD)
	ee.exf_xplink = false
	ee.exi_offset = uint32(_exi[ppa4_exi_ix].exi_length)

	goc_ref_exi_ix := getNextExiIx()
	ee = getNewExiEntry(goc_ref_exi_ix)
	addRefExiData(ee, "GO#C", _ccsect_exi_ix, EXT_ER)
	ee.exf_xplink = true
	ee.exf_weak_ref = true

	offset := int32(_exi[ppa4_exi_ix].exi_length)
	addRli(goc_ref_exi_ix, ppa4_exi_ix, SIZE_ADDR_CON_32, RS_NEG, offset)

	// Code size
	_ppa4._codeSize = uint64(offset + int32(unsafe.Sizeof(_ppa4)))

	// Add TXI for PPA4 (compile unit metadata)
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, &_ppa4)
	ppa4_mem := *(*[]byte)(unsafe.Pointer(&bin_buf))

	addTxi(ppa4_exi_ix, 0, int64(len(ppa4_mem)), &ppa4_mem, TXTTS_BYTE)
}

func buildCELQSTRT() {

	_celqstrt_exi_ix = getNextExiIx()
	ee := getNewExiEntry(_celqstrt_exi_ix)
	addDefExiCode(ee, "CELQSTRT", 0, EXT_SD)
	ee.exf_xplink = false
	ee.exf_force_rent = true

	_start_code_exi_ix = getNextExiIx()
	ee = getNewExiEntry(_start_code_exi_ix)
	addDefExiCode(ee, "G_CODE64", _celqstrt_exi_ix, EXT_ED)
	ee.exi_alignment = EXIAL_DWORD
	ee.exf_xplink = false

	ld_exi_ix := getNextExiIx()
	ee = getNewExiEntry(ld_exi_ix)
	addDefExiCode(ee, "CELQSTRT", _start_code_exi_ix, EXT_LD)
	ee.exf_xplink = false

	celqmain_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqmain_exi_ix)
	addRefExiData(ee, "CELQMAIN", _celqstrt_exi_ix, EXT_ER)
	ee.exf_xplink = false
	ee.exf_weak_ref = true

	celqfman_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqfman_exi_ix)
	addRefExiData(ee, "CELQFMAN", _celqstrt_exi_ix, EXT_ER)
	ee.exf_xplink = false
	ee.exf_weak_ref = true

	celqetbl_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqetbl_exi_ix)
	addRefExiData(ee, "CELQETBL", _celqstrt_exi_ix, EXT_ER)
	ee.exf_xplink = false

	celqllst_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqllst_exi_ix)
	addRefExiData(ee, "CELQLLST", _celqstrt_exi_ix, EXT_ER)
	ee.exf_xplink = false

	celqbst_exi_ix := getNextExiIx()
	ee = getNewExiEntry(celqbst_exi_ix)
	addRefExiCode(ee, "CELQBST", _celqstrt_exi_ix, EXT_ER)
	ee.exf_xplink = false

	// construct the CELQSTRT binary code
	var _celqstrt_buffer celqstrt_text

	// section 2
	_celqstrt_buffer.nop_1 = 0x47000000 // 00:   NOOP  0
	_celqstrt_buffer.nop_2 = 0x47000002 // 04:   NOOP  2
	_celqstrt_buffer.stmg = 0xEBECD008  // 08:   STMG  r14,r12,8(r13)
	_celqstrt_buffer.stmg_p2 = 0x0024
	_celqstrt_buffer.bru1 = 0xA7F4 // 0E:   BRU AROUND
	_celqstrt_buffer.bru2 = 0x000E

	// SIGNATUR BRU *
	_celqstrt_buffer.dcsiglen = 0x0018                   // 12:   AL2(AROUND-SIGNATUR)
	_celqstrt_buffer.dcsignature = 0xCE010000 + LE_VV_RR // 14:   'CE' and 64 bit  and LE version release
	_celqstrt_buffer.adparamlist = 0x0000000000000038    // 18: AD(Param List)
	_celqstrt_buffer.signeye = 0xC3C5C5E2E3C1D9E3        // 20: CEESTART eyecatcher
	_celqstrt_buffer.xplmainreserve = 0x0100             // 28:   XPLINK main + reserved (0)

	// section 3
	_celqstrt_buffer.balr = 0x0530   // 2A:   BALR r3,r0
	_celqstrt_buffer.lg = 0xE3F03064 // 2C:   LG r15,AD(CELQBST)
	_celqstrt_buffer.lg_p2 = 0x0004
	_celqstrt_buffer.balr2 = 0x050F // 32:   BALR r0,r15

	// section 4
	_celqstrt_buffer.versionmark = 0xFFFD             // 40:   -3
	_celqstrt_buffer.al2stlen = 0x0058                // 42:   AL2(STLEN)
	_celqstrt_buffer.adsignature = 0x0000000000000012 // 60: AD(SIGNATUR)

	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, _celqstrt_buffer)
	mem := *(*[]byte)(unsafe.Pointer(&bin_buf))

	_celqstrt_offset = _cummulativeTXTSize

	addTxi(_start_code_exi_ix, 0, int64(unsafe.Sizeof(_celqstrt_buffer)), &mem, TXTTS_BYTE)
	//	fmt.Printf("celqstrt: %08x\n", _start_code_exi_ix)
	//	fmt.Printf("length: %08x\n", _exi[_start_code_exi_ix].exi_length)

	addRli(ld_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_POS, int32(unsafe.Offsetof(_celqstrt_buffer.adparamlist)))
	addRli(ld_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_POS, int32(unsafe.Offsetof(_celqstrt_buffer.adsignature)))
	addRli(celqmain_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, int32(unsafe.Offsetof(_celqstrt_buffer.adcelqmain)))
	addRli(celqfman_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, int32(unsafe.Offsetof(_celqstrt_buffer.adcelqfman)))
	addRli(celqetbl_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, int32(unsafe.Offsetof(_celqstrt_buffer.adcelqetbl)))
	addRli(celqllst_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, int32(unsafe.Offsetof(_celqstrt_buffer.adcelqllst)))
	addRli(celqbst_exi_ix, _start_code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, int32(unsafe.Offsetof(_celqstrt_buffer.adcelqbst)))

}

func getCEESTART_ERExi() uint32 {
	if _start_er_exi_ix == 0 {
		// "CEESTART": ER reference
		_start_er_exi_ix = getNextExiIx()
		ee := getNewExiEntry(_start_er_exi_ix)
		addRefExiCode(ee, "CELQSTRT", _ccsect_exi_ix, EXT_ER)
		ee.exf_xplink = false
	}
	return _start_er_exi_ix
}

func buildCELQMAIN() {

	// SD: CELQMAIN
	ceemain_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ceemain_exi_ix)
	addDefExiCode(ee, "CELQMAIN", 0, EXT_SD)
	ee.exf_xplink = false
	ee.exf_executable = false

	// ED: CELQMAIN
	data_exi_ix := getNextExiIx()
	ee = getNewExiEntry(data_exi_ix)
	addDefExiCode(ee, "G_CODE64", ceemain_exi_ix, EXT_ED)
	ee.exi_length = 0
	ee.exi_alignment = EXIAL_QWORD
	ee.exf_xplink = false
	ee.exf_executable = false
	ee.exf_force_rent = true

	// LD: CELQMAIN
	ee = getNewExiEntry(getNextExiIx())
	addDefExiCode(ee, "CELQMAIN", data_exi_ix, EXT_LD)
	ee.exf_xplink = false
	ee.exf_executable = false

	// ER: CELQINPL
	edcinpl_ref_exi_ix := getNextExiIx()
	ee = getNewExiEntry(edcinpl_ref_exi_ix)
	addRefExiCode(ee, "CELQINPL", ceemain_exi_ix, EXT_ER)
	ee.exf_xplink = false

	// ER: main entry
	main_ref_exi_ix := getNextExiIx()
	ee = getNewExiEntry(main_ref_exi_ix)
	ee.exf_mapped = true
	addRefExiCode(ee, INITENTRY, _code_exi_ix, EXT_ER)
	ee.exf_mapped = false
	ee.exf_xplink = true

	// CELQMAIN: text and relocations
	var _celqmain_buffer celqmain_text_xplink

	_celqmain_buffer.first_word = 0x0500000100000000 //norent for now
	_celqmain_buffer.admain = 0

	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, _celqmain_buffer)
	mem := *(*[]byte)(unsafe.Pointer(&bin_buf))

	addTxi(data_exi_ix, 0, int64(len(mem)), &mem, TXTTS_BYTE)

	addRli(edcinpl_ref_exi_ix, data_exi_ix, SIZE_ADDR_CON_64,
		RS_NONE, int32(unsafe.Offsetof(_celqmain_buffer.adcelqinpl)))
	addRli(main_ref_exi_ix, data_exi_ix, SIZE_ADDR_CON_64,
		RS_POS, int32(unsafe.Offsetof(_celqmain_buffer.admain)))

	/*
	   if _static_exi_ix > 0 { // RENT
	      Diag(" Should not be RENT \n")
	      _celqmain_buffer.a0 = 0x00000000
	      addRli(main_ref_exi_ix, data_exi_ix, SIZE_ADDR_CON_32,
	             RS_ADA, int32 (unsafe.Offsetof(_celqmain_buffer.qenv)))
	   } else {
	      _celqmain_buffer.adenv = 0xFFFFFFFFFFFFFFFF
	   }
	*/
}

func buildPPA2Chain() {
	var data_exi_ix uint32
	var _ppa2chain_text [2]uint32

	// ED
	ppa2_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ppa2_exi_ix)
	addDefExiData(ee, "C_@@QPPA2", _ccsect_exi_ix, EXT_ED)
	ee.exf_merge = true
	ee.exf_force_rent = true
	ee.exi_alignment = EXIAL_DWORD

	// PR
	pr_exi_ix := getNextExiIx()
	ee = getNewExiEntry(pr_exi_ix)
	addDefExiData(ee, " ", ppa2_exi_ix, EXT_PR)
	ee.exf_merge = true
	ee.exf_mapped = false
	ee.exf_xplink = false
	ee.exi_length = 0
	data_exi_ix = pr_exi_ix

	_ppa2Offset = uint64(_ppa2Buffer._ceestartOffset)

	if _ppa2Buffer._ceestartOffset < 0 {
		temp := -_ppa2Buffer._ceestartOffset
		_ppa2Offset = uint64(temp)
	} else {
		_ppa2Offset = uint64(_ppa2Buffer._ceestartOffset)
	}

	_ppa2chain_text[0] = uint32((_ppa2Offset >> 32) & 0xFFFFFFFF)
	_ppa2chain_text[1] = uint32((_ppa2Offset & 0xFFFFFFFF))

	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, _ppa2chain_text)
	mem := *(*[]byte)(unsafe.Pointer(&bin_buf))

	addTxi(data_exi_ix, 0, 8, &mem, TXTTS_BYTE)
	addRli(_ccsect_rld_exi_ix, data_exi_ix, SIZE_ADDR_CON_64, RS_POS, 0)
	addRli(getCEESTART_ERExi(), data_exi_ix, SIZE_ADDR_CON_64, RS_NEG, 0)
}

func assignExiIxToSection(ds *dwarf_section, length int64) {

	alignment := uint32(8)
	var merge bool

	// "ED" for section
	ed_exi_ix := getNextExiIx()
	ee := getNewExiEntry(ed_exi_ix)
	addDefExiData(ee, ds.name, _ccsect_exi_ix, EXT_ED)
	ee.exf_def = true
	ee.exf_mapped = true
	ee.exf_xplink = false
	ee.exf_noload = true
	ee.exf_removable = false // gc->getRemovable()
	ee.exi_flag = ESDTS_BYTE << 4

	merge = false
	ee.exi_length = 0
	ee.exf_merge = merge
	ee.exi_alignment = alignment

	// "LD" or "PR"
	exi_ix := getNextExiIx()
	ee = getNewExiEntry(exi_ix)
	ds.refESD = exi_ix
	tempstr := "#"
	tempstr += ds.name

	addDefExiData(ee, tempstr, ed_exi_ix, EXT_LD)
	ds.defESD = ed_exi_ix // ED for text

	ee.exi_alignment = alignment
	ee.exf_merge = merge
	ee.exf_xplink = false
	ee.exf_def = true
	ee.exf_mapped = true
	ee.exf_xplink = false
	ee.exf_noload = true
	ee.exi_flag = ESDTS_BYTE << 4

}

func newzOSDWARFSection(name string, size int64, buf []byte) {
	if size == 0 {
		return
	}

	var ds *dwarf_section
	newBuf := buf

	ds = getDwarfSection(name)

	/*
	   	fmt.Printf(" debug section %s size = 0x%x %d\n",name, size,size)
	   	fmt.Printf(" 00:\t")
	           for i := int64(0) ; i < size; i++ {
	                   fmt.Printf(" %x ",newBuf[i])
	                   if (i+1) % 4 == 0 {
	                           fmt.Printf(" \n %x\t",i+1)
	                   }
	           }
	*/

	assignExiIxToSection(ds, size)
	addTxi(ds.defESD, 0, size, &newBuf, TXTTS_BYTE)
}

func getDwarfSection(name string) *dwarf_section {
	switch name {
	case ".debug_abbrev":
		return &_debug_abbrev
	case ".debug_line":
		return &_debug_line
	case ".debug_frame":
		return &_debug_frame
	case ".debug_info":
		return &_debug_info
	case ".debug_pubnames":
		return &_debug_pubnames
	case ".debug_pubtypes":
		return &_debug_pubtypes
	case ".debug_aranges":
		return &_debug_aranges
	default:
		Diag("debug section not supported on this platform\n")
	}
	return nil
}

func buildDebugParts() {

	Dwarfemitdebugsections()

	_debug_line.sec = linesec
	_debug_frame.sec = framesec
	_debug_info.sec = infosec
	_debug_aranges.sec = arangessec

	var ds *dwarf_section

	ds = _debugSectionList

	for ; ds != nil; ds = ds.next {
		if ds.sec != nil && int64(len(ds.sec.R)) > 0 {
			for i := 0; i < len(ds.sec.R); i++ {
				r := &ds.sec.R[i]
				pESD := ds.defESD
				poffset := int32(_exi[pESD].exi_offset) + r.Off
				var r1ESD uint32

				// QCON for .debug section
				if strings.HasPrefix(r.Sym.Name, ".debug") {
					sect := getDwarfSection(r.Sym.Name)
					r1ESD = sect.refESD
					addRli_T(r1ESD, pESD, 4, RS_POS, poffset, RLT_QCON)
				} else {
					r1ESD = _shash[r.Sym.Value]
					if r1ESD != 0 {
						addRli(r1ESD, pESD, SIZE_ADDR_CON_64, RS_NONE, poffset)
					}
				}
				/*
				   fmt.Printf(" buildRelocation pptr = %s pESD = 0x%x r.Off = 0x%x\n", ds.name ,pESD,r.Off)
				   fmt.Printf("                 r1ptr = %s r1ESD = 0x%x \n", r.Sym.Name, r1ESD)
				*/
			}
		}

	}
}

func buildRelocation() {

	var pESD uint32
	var r1ESD uint32

	for i := range _objectCodeRelocationList {
		r := _objectCodeRelocationList[i]
		pESD = _shash[r._pptr.Value]
		r1ESD = _shash[r._r1ptr.Value]

		poffset := int32(_exi[pESD].exi_offset) + r._poff

		/*
			fmt.Printf(" buildRelocation pptr = %s type = %d  addr = 0x%x  offset = 0x%x pESD = %d \n", r._pptr.Name, r._pptr.Type, Symaddr(r._pptr),r._poff,pESD)
			fmt.Printf(" buildRelocation r1ptr = %s type = %d  addr = 0x%x offset = 0x%x  value = 0x%x r1ESD = %d 0x%x \n", r._r1ptr.Name, r._r1ptr.Type, Symaddr(r._r1ptr),r._r1off,r._r1ptr.Value,r1ESD,r1ESD)
		*/
		// chwan -
		// Add two RLI entries ADA+EP.
		if r._type == ObjectCodeADARelocation {
			ref_exi_ix := getNextExiIx()
			ee := getNewExiEntry(ref_exi_ix)
			addRefExiCode(ee, r._r1ptr.Name, _code_exi_ix, EXT_ER)
			ee.exf_xplink = true
			addRli(ref_exi_ix, _code_exi_ix, SIZE_ADDR_CON_64, RS_EP, poffset+8)
			addRli(ref_exi_ix, _code_exi_ix, SIZE_ADDR_CON_64, RS_ADA, poffset)
		} else if r1ESD != 0 && Symaddr(r._r1ptr) != 0 && r._r1ptr.Value != 0 {
			addRli(r1ESD, _code_exi_ix, SIZE_ADDR_CON_64, RS_NONE, poffset)
		}
	}

}

func goffwriteHdr() uint32 {
	a := uint32(0)
	Cput(ghdr.hdr_ptv_prefix)
	a += 1
	Cput(ghdr.hdr_ptv_flag)
	a += 1
	Cput(ghdr.hdr_ptv_version)
	a += 1
	Cput(ghdr.hdr_reserved_1)
	a += 1
	Thearch.Lput(uint32(ghdr.hdr_hardware_env))
	a += 4
	Thearch.Lput(uint32(ghdr.hdr_os))
	a += 4
	Thearch.Lput(uint32(ghdr.hdr_CCSID))
	a += 4

	for i := 0; i < 16; i++ {
		Cput(ghdr.hdr_char_set_name[i])
		a += 1
	}

	for i := 0; i < 16; i++ {
		Cput(ghdr.hdr_lang_prod_id[i])
		a += 1
	}

	Thearch.Lput(uint32(ghdr.hdr_arch_level))
	a += 4
	Thearch.Wput(uint16(ghdr.hdr_mod_properties_len))
	a += 2

	for i := 0; i < 6; i++ {
		Cput(ghdr.hdr_reserved_2[i])
		a += 1
	}

	Thearch.Wput(ghdr.hdr_internal_CCSID)
	a += 2
	Thearch.Wput(ghdr.hdr_software_env)
	a += 2

	for i := 0; i < 16; i++ {
		Cput(ghdr.hdr_reserved_3[i])
		a += 1
	}

	i := a % 80
	for i > 0 && i < 80 {
		Cput('0')
		a += 1
		i++
	}
	return a

}

func writeObjectRecord(record goff_esd_record) uint32 {
	a := uint32(0)
	Cput(record.esd_ptv_prefix)
	a += 1
	Cput(record.esd_ptv_flag1)
	a += 1
	Cput(record.esd_ptv_version)
	a += 1
	Cput(record.esd_symbol_type)
	a += 1
	Thearch.Lput(uint32(record.esd_esdid))
	a += 4
	Thearch.Lput(uint32(record.esd_parent_esdid))
	a += 4
	Thearch.Lput(uint32(record.esd_reserved_1))
	a += 4
	Thearch.Lput(uint32(record.esd_offset))
	a += 4
	Thearch.Lput(uint32(record.esd_reserved_2))
	a += 4
	Thearch.Lput(uint32(record.esd_length))
	a += 4
	Thearch.Lput(uint32(record.esd_ext_attr_esdid))
	a += 4
	Thearch.Lput(uint32(record.esd_ext_attr_offset))
	a += 4
	Thearch.Lput(uint32(record.esd_alias))
	a += 4
	Cput(record.esd_name_space_id)
	a += 1
	Cput(record.esd_ptv_flag2)
	a += 1
	Cput(record.esd_fill_byte_value)
	a += 1
	Cput(record.esd_reserved_4)
	a += 1
	Thearch.Lput(uint32(record.esd_ada_esdid))
	a += 4
	Thearch.Lput(uint32(record.esd_sort_priority))
	a += 4

	for i := 0; i < 8; i++ {
		Cput(record.esd_signature[i])
		a += 1
	}

	Cput(record.esd_amode)
	a += 1
	Cput(record.esd_rmode)
	a += 1
	Cput(record.esd_ptv_flag3)
	a += 1
	Cput(record.esd_ptv_flag4)
	a += 1
	Cput(record.esd_ptv_flag5)
	a += 1
	Cput(record.esd_ptv_flag6)
	a += 1
	Cput(record.esd_ptv_flag7)
	a += 1

	for i := 0; i < 3; i++ {
		Cput(record.esd_reserved_6[i])
		a += 1
	}

	Thearch.Wput(uint16(record.esd_name_length))
	a += 2

	if record.esd_name_length > 0 {
		for i := int16(0); i < 8 && i < record.esd_name_length; i++ {
			Cput(record.esd_name[i])
			a += 1
		}
	}

	i := a % 80
	for i > 0 && i < 80 {
		Cput('0')
		a += 1
		i++
	}
	return a
}

func writeContinuationRecord(t uint8, data *[]byte, offset int, length int) uint32 {
	var record goff_continuation_record
	a := uint32(0)

	for i := length; i > 0; {
		j := length - i
		record.cont_ptv_prefix = GOFF_PTV_PREFIX
		record.cont_ptv_flag = 0
		record.cont_ptv_flag = t & 0xF0                        // set cont_ptv_type
		record.cont_ptv_flag = record.cont_ptv_flag | (1 << 1) // cont_ptv_continuation = true
		for k := 0; k < len(record.cont_data); k++ {
			record.cont_data[k] = '0'
		}
		if i > len(record.cont_data) {
			record.cont_ptv_flag = record.cont_ptv_flag | 1 //  cont_ptv_continued = true
			for k := 0; k < len(record.cont_data); k++ {
				record.cont_data[k] = (*data)[offset+j+k]
			}
		} else {
			record.cont_ptv_flag = record.cont_ptv_flag & 0xFE //  cont_ptv_continued = false
			for k := 0; k < i; k++ {
				record.cont_data[k] = (*data)[offset+j+k]
			}
		}

		//  writeObjectRecord((uint8_t *)&record, sizeof(record));
		Cput(record.cont_ptv_prefix)
		a += 1
		Cput(record.cont_ptv_flag)
		a += 1
		Cput(record.cont_ptv_version)
		a += 1

		for k := 0; k < len(record.cont_data); k++ {
			Cput(record.cont_data[k])
			a += 1
		}

		i -= len(record.cont_data)
	}

	return a
}

func writeTxtRecord(record goff_txt_record) uint32 {
	a := uint32(0)
	Cput(record.txt_ptv_prefix)
	a += 1
	Cput(record.txt_ptv_flag1)
	a += 1
	Cput(record.txt_ptv_version)
	a += 1
	Cput(record.txt_ptv_flag2)
	a += 1

	Thearch.Lput(record.txt_element_esdid)
	a += 4
	Thearch.Lput(uint32(record.txt_reserved_2))
	a += 4
	Thearch.Lput(uint32(record.txt_offset))
	a += 4
	Thearch.Lput(uint32(record.txt_true_length))
	a += 4

	Thearch.Wput(uint16(record.txt_encoding_type))
	a += 2
	Thearch.Wput(record.txt_data_length)
	a += 2

	// And now the variable part...
	for i := 0; i < len(record.txt_data) && i < TDFIXEDSIZE; i++ {
		Cput(uint8(record.txt_data[i]))
		a += 1
	}

	if a != 80 {
		Diag("Record size is not fixed to 80\n")
	}
	return a
}

func writeESDFromExi(ee exi_entry) uint32 {
	var record goff_esd_record

	a := uint32(0)
	name_len := len(ee.exi_name)

	record.esd_ptv_prefix = GOFF_PTV_PREFIX
	// record.esd_ptv_type = GOFF_ESD
	record.esd_ptv_flag1 = GOFF_ESD << 4
	record.esd_esdid = int32(ee.exi_esdid)

	switch ee.exi_type {
	case EXT_SD:
		record.esd_symbol_type = ESDTY_SD
		if ee.exf_executable {
			record.esd_ptv_flag4 = ESDTA_RENT << 5 // esd_tasking_behaviour field
		}
		break

	case EXT_ED:
		record.esd_symbol_type = ESDTY_ED
		if ee.exf_removable {
			record.esd_ptv_flag2 = 0xFF & 0x10 // esd_removable field
		}

		if ee.exf_executable {
			record.esd_ptv_flag4 = 0xFF & ESDEX_INSTR // esd_executable field
			record.esd_ptv_flag4 = 0xFF & 0x08        // esd_read_only
		} else {
			if ee.exf_execunspecified {
				record.esd_ptv_flag4 = 0xFF & ESDEX_UNSPECIFIED // esd_executable
			} else {
				record.esd_ptv_flag4 = 0xFF & ESDEX_DATA // esd_executable
			}
			if ee.exf_force_rent || ee.exf_readonly {
				record.esd_ptv_flag4 = 0xFF & 0x08a // esd_read_only
			}
		}

		record.esd_offset = int32(ee.exi_offset)
		record.esd_length = int32(ee.exi_length)
		record.esd_ptv_flag7 = 0xFF & uint8(ee.exi_alignment) // esd_alignment
		record.esd_ptv_flag2 = 0xFF & 0x80                    // esd_fill_byte_present
		record.esd_fill_byte_value = 0
		// record.esd_name_mangled = 0   // to be finalized
		record.esd_amode = _amode // THIS IS NOT NECESSARY ... see
		// MVS Program Management: Advanced Facilities...
		// ED takes residency property but not addressing property
		record.esd_rmode = _rmode
		record.esd_ptv_flag3 = 0xFF & ee.exi_flag // esd_text_rec_style = ee.exi_rec_style
		if ee.exf_merge {
			record.esd_ptv_flag3 = 0xFF & ESDBA_MERGE // esd_binding_algorithm
		} else {
			record.esd_ptv_flag3 = 0xFF & ESDBA_CONCAT // esd_binding_algorithm
		}
		if ee.exf_deferred {
			record.esd_ptv_flag6 = 0xFF & (ESDCL_DEFERRED << 6) // esd_loading_behaviour
		}
		if ee.exf_noload {
			record.esd_ptv_flag6 = 0xFF & (ESDCL_NOLOAD << 6) // esd_loading_behaviour
		}
		if ee.exf_c_wsa {
			record.esd_ptv_flag2 = 0xFF & ESDRQ_1 // esd_ed_reserve_qwords
		} else {
			record.esd_ptv_flag2 = 0xFF & ESDRQ_0
		}
		if ee.exf_force_rent {
			record.esd_ptv_flag4 = 0xFF & 0x08a // esd_read_only
		}
		record.esd_parent_esdid = int32(_exi[ee.exi_parent_exi_ix].exi_esdid)
		record.esd_name_space_id = uint8(ee.exi_namespace)
		break

	case EXT_LD:
		record.esd_symbol_type = ESDTY_LD
		if ee.exf_executable {
			record.esd_ptv_flag4 = 0xFF & ESDEX_INSTR // esd_executable
		} else {
			record.esd_ptv_flag4 = 0xFF & ESDEX_DATA // esd_executable
		}
		if ee.exf_weak_def {
			record.esd_ptv_flag5 = 0xFF & ESDST_WEAK // esd_binding_strength
		} else {
			record.esd_ptv_flag5 = 0xFF & ESDST_STRONG // esd_binding_strength
		}
		if ee.exf_xplink {
			record.esd_ptv_flag7 = 0xFF & 0x20 // esd_linkage_xplink

			if _static_exi_ix > 0 && ee.exf_executable {
				record.esd_ada_esdid = int32(_exi[_static_exi_ix].exi_esdid)
			}
		}
		if !ee.exf_mapped {
			record.esd_ptv_flag2 = 0xFF & 0x20 // esd_sym_renamable
		}
		record.esd_offset = int32(ee.exi_offset)
		record.esd_amode = _amode
		record.esd_name_space_id = uint8(ee.exi_namespace)
		record.esd_ptv_flag6 = 0xFF & setBindingScope(ee) // esd_binding_scope
		record.esd_parent_esdid = int32(_exi[ee.exi_parent_exi_ix].exi_esdid)
		break

	case EXT_ER:
		record.esd_symbol_type = ESDTY_ER
		if ee.exf_executable {
			record.esd_ptv_flag4 = 0xFF & ESDEX_INSTR // esd_executable
		} else {
			record.esd_ptv_flag4 = 0xFF & ESDEX_DATA // esd_executable
		}
		if ee.exf_weak_ref {
			record.esd_ptv_flag5 = 0xFF & ESDST_WEAK // esd_binding_strength
		} else {
			record.esd_ptv_flag5 = 0xFF & ESDST_STRONG // esd_binding_strength
		}
		if ee.exf_xplink {
			record.esd_ptv_flag7 = 0xFF & 0x20 // esd_linkage_xplink
		}
		if !ee.exf_mapped {
			record.esd_ptv_flag2 = 0xFF & 0x20 // esd_sym_renamable
		}
		if ee.exf_indirect {
			record.esd_ptv_flag6 = 0xFF & 0x10 // esd_indirect_reference
		}
		record.esd_offset = int32(ee.exi_offset)
		record.esd_name_space_id = uint8(ee.exi_namespace)
		record.esd_ptv_flag2 = 0xFF & ESDES_NONE          // esd_er_symbol_type
		record.esd_ptv_flag6 = 0xFF & setBindingScope(ee) // esd_binding_scope
		record.esd_parent_esdid = int32(_exi[ee.exi_parent_exi_ix].exi_esdid)
		record.esd_amode = _amode
		break

	case EXT_PR:
		record.esd_symbol_type = ESDTY_PR
		if ee.exf_executable {
			record.esd_ptv_flag4 = 0xFF & ESDEX_INSTR // esd_executable
		} else {
			record.esd_ptv_flag4 = 0xFF & ESDEX_DATA // esd_executable
		}
		record.esd_name_space_id = uint8(ee.exi_namespace)
		record.esd_ptv_flag7 = 0xFF & uint8(ee.exi_alignment) // esd_alignment
		record.esd_length = int32(ee.exi_length)
		record.esd_amode = _amode // THIS IS NOT NECESSARY ... see
		// MVS Program Management: Advanced Facilities...
		// PR takes neither residency property nor addressing property
		record.esd_parent_esdid = int32(_exi[ee.exi_parent_exi_ix].exi_esdid)
		if ee.exf_xplink {
			record.esd_ptv_flag7 = 0xFF & 0x20 // esd_linkage_xplink
		}
		record.esd_ptv_flag6 = 0xFF & setBindingScope(ee) // esd_binding_scope
		if !ee.exf_mapped {
			record.esd_ptv_flag2 = 0xFF & 0x20 // esd_sym_renamable
		}
		if ee.exf_weak_ref {
			record.esd_ptv_flag5 = 0xFF & (MDEF_NO_WARNING << 5) // esd_mdef
		} else {
			record.esd_ptv_flag5 = 0xFF & (MDEF_WARNING << 5) // esd_mdef
		}
		if ee.exf_indirect { // DLL function descriptors
			record.esd_ptv_flag6 = 0xFF & 0x10 // esd_indirect_reference
		}
		record.esd_sort_priority = int32(ee.exi_sort_priority)
		break
	default:
	}

	record.esd_name_length = int16(name_len)
	if name_len >= 32767 {
		Diag("name length exceeds maximum for GOFF")
	}
	if name_len <= 8 {
		record.esd_ptv_flag1 = 0xFD & record.esd_ptv_flag1 // esd_ptv_continuation = false
		record.esd_ptv_flag1 = 0xFE & record.esd_ptv_flag1 // esd_ptv_continued = false
		record.esd_name = ee.exi_name
		a = writeObjectRecord(record)
	} else {
		record.esd_ptv_flag1 = 0xFD & record.esd_ptv_flag1 // esd_ptv_continuation = false
		record.esd_ptv_flag1 = record.esd_ptv_flag1 | 1    // esd_ptv_continued = true
		record.esd_name = ee.exi_name
		a = writeObjectRecord(record)
		b := []byte(ee.exi_name)
		a += writeContinuationRecord(record.esd_ptv_flag1, &b, 8, name_len-8)
	}
	return a
}

func goffwriteESDsFromExis() uint32 {
	a := uint32(0)
	for i := uint32(0); i <= _next_exi_ix; i++ {
		a += writeESDFromExi(_exi[i])
	}
	return a
}

func goffwriteTXTFromTxi(txi_ix uint32) uint32 {
	te := _txi[txi_ix]
	data_len := te.txi_length
	data_offset := te.txi_offset
	a := uint32(0)
	offset := TDFIXEDSIZE

	// Trim the TXT of any trailing zeros.  We know that the owning SD/ED/PR
	// will have set "esd_fill_byte_present" to TRUE, and will have
	// supplied the "esd_fill_byte_value" of binary zero, so that the Binder
	// will pad any too-short record (i.e. shorter than the length specified
	// on the associated SD/ED/PR) with the missing binary zeros.

	/*
	   if  (*data_ptr)[0] == 0 {
	      Diag ("Encounter trailing zero in data_ptr \n")
	   }
	*/

	// The length of the txt_data is limited to MAX_VAR_DATA_LEN.  This
	// loop handles this situation by writing multiple TXT cards (and
	// continuations) as necessary so that each has its txt_data_length a
	// maximum of MAX_VAR_DATA_LEN.
	doff := 0 // data offset

	for data_len > 0 {
		var record goff_txt_record
		var t_len uint32
		if data_len > MAX_VAR_DATA_LEN {
			t_len = MAX_VAR_DATA_LEN
		} else {
			t_len = data_len
		}

		record.txt_ptv_prefix = GOFF_PTV_PREFIX
		record.txt_ptv_flag1 = GOFF_TXT << 4 // txt_ptv_type

		//    record.txt_rec_style = te.txi_rec_style;
		record.txt_ptv_flag2 = record.txt_ptv_flag2 | (te.txi_flag >> 4)
		record.txt_element_esdid = _exi[te.txi_exi_ix].exi_esdid
		record.txt_offset = data_offset
		record.txt_data_length = uint16(t_len)

		if data_len <= uint32(len(record.txt_data)) {
			record.txt_ptv_flag1 = 0xFD & record.txt_ptv_flag1 //txt_ptv_continuation = false
			record.txt_ptv_flag1 = 0xFE & record.txt_ptv_flag1 //txt_ptv_continued = false

			for i := 0; i < len(record.txt_data) && i < int(record.txt_data_length); i++ {
				record.txt_data[i] = (*te.txi_data_ptr)[doff]
				doff++
			}
			a = writeTxtRecord(record)
		} else {
			record.txt_ptv_flag1 = 0xFD & record.txt_ptv_flag1 // record.txt_ptv_flag1 //txt_ptv_continuation = false
			record.txt_ptv_flag1 = record.txt_ptv_flag1 | 1    // txt_ptv_continued = true
			for i := 0; i < len(record.txt_data); i++ {
				record.txt_data[i] = (*te.txi_data_ptr)[doff]
				doff++

			}
			a = writeTxtRecord(record)
			a += writeContinuationRecord(record.txt_ptv_flag1, te.txi_data_ptr, offset, int(t_len-TDFIXEDSIZE))
			doff += (MAX_VAR_DATA_LEN - TDFIXEDSIZE) // 16328
		}

		data_len = data_len - t_len
		data_offset = data_offset + int32(t_len)
		offset += int(t_len)
	}
	return a
}

func goffwriteTXTsFromTxis() uint32 {
	a := uint32(0)

	if _next_txi_ix > 0 {
		for i := uint32(0); i <= _next_exi_ix; i++ {
			for txi_ix := _exi[i].exi_first_txi_ix; txi_ix > 0; {
				if _txi[txi_ix].txi_length > 0 {
					a += goffwriteTXTFromTxi(txi_ix)
				}
				txi_ix = _txi[txi_ix].txi_next_ix
			}
		}
	}
	return a
}

func goffwriteRLDsFromRlis() uint32 {
	a := uint32(0)
	var rdata goff_rld_data
	var current_pattern uint32
	var rdi_ptr *goff_rld_data

	current_pid := uint32(0)
	current_rid := uint32(0)
	current_offset := uint32(0)
	previous_pid := uint32(0)
	previous_rid := uint32(0)
	previous_offset := uint32(0)

	rdi_ptr = &rdata
	for i := uint32(1); i <= _next_rli_ix; i++ {
		current_pattern = NONE

		rdi_ptr.rld_data = new(goff_rld_data_item)
		switch _rli[i].rli_type {
		case RLT_ADCON:
			if _rli[i].rli_sign == RS_NEG {
				rdi_ptr.rld_data.rld_data_item_byte2 |= 1 << 1
			}
			break
		case RLT_VCON:
			// chwan - tobey sets this to true. But somehow we can't do this to GO.
			//         But we need it to be true for the XPLINK function descriptor part.
			//         This is the reason we created RLT_XVCON for it.
			rdi_ptr.rld_data.rld_data_item_byte2 = 0 //  rld_no_fetch_fixup_field = false
			break
		case RLT_CONDVCON:
			rdi_ptr.rld_data.rld_data_item_byte2 = 0      //  rld_no_fetch_fixup_field = false
			rdi_ptr.rld_data.rld_data_item_byte5 = 1 << 4 // rld_condseq = true
			break
		case RLT_QCON:
			rdi_ptr.rld_data.rld_data_item_byte1 = RLDRT_R_OFFSET << 4 // rld_ref_type = RLDRT_R_OFFSET
			rdi_ptr.rld_data.rld_data_item_byte1 |= RLDRO_CLASS        // rld_ref_origin = RLDRO_CLASS
			if _rli[i].rli_sign == RS_NEG {
				rdi_ptr.rld_data.rld_data_item_byte2 |= RLDAC_SUB << 1 // rld_action = RLDAC_SUB
			}
			break
		case RLT_ADA:
			rdi_ptr.rld_data.rld_data_item_byte1 = RLDRT_R_ADA << 4 // rld_ref_type = RLDRT_R_ADA
			// chwan - this has to be set to true to make the Binder happy.
			//rdi_ptr.rld_data.rld_data_item_byte2 = 0      //  rld_no_fetch_fixup_field = false
			rdi_ptr.rld_data.rld_data_item_byte2 = 1
			break
		case RLT_XVCON:
			// chwan - see above
			rdi_ptr.rld_data.rld_data_item_byte2 = 1 //  rld_no_fetch_fixup_field = true
			break
		case RLT_RI:
			rdi_ptr.rld_data.rld_data_item_byte1 = RLDRT_RI_REL << 4 // rld_ref_type = RLDRT_RI_REL
			if _rli[i].rli_sign == RS_NEG {
				rdi_ptr.rld_data.rld_data_item_byte2 |= 1 << 1
			}
			break
		case RLT_REL:
			rdi_ptr.rld_data.rld_data_item_byte1 = RLDRT_R_REL << 4 // rld_ref_type = RLDRT_R_REL
			if _rli[i].rli_sign == RS_NEG {
				rdi_ptr.rld_data.rld_data_item_byte2 |= RLDAC_SUB << 1 // rld_action = RLDAC_SUB
			}
		default:
		}

		current_pid = _exi[_rli[i].rli_in_exi_ix].exi_esdid
		current_rid = _exi[_rli[i].rli_ref_exi_ix].exi_esdid
		current_offset = uint32(_rli[i].rli_offset)

		// Determine if we have any repeating paterns that allows us
		// to use a compressed RLD form
		if i > 1 {
			if current_pid == previous_pid {
				current_pattern |= SAME_PID
			}
			if current_rid == previous_rid {
				current_pattern |= SAME_RID
			}
			if current_offset == previous_offset {
				current_pattern |= SAME_OFFSET
			}
		}

		rdi_ptr.rld_data.rld_targ_field_byte_len = uint8(_rli[i].rli_length)

		var size int32

		// Write out the RLD data depending on repetition pattern
		switch current_pattern {
		case NONE:
			rdi_ptr.rdiRPOff_ptr = new(goff_rld_data_item_RPOff)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiRPOff_ptr))
			rdi_ptr.rld_type = NONE
			rdi_ptr.rdiRPOff_ptr.rld_R_esdid = int32(_exi[_rli[i].rli_ref_exi_ix].exi_esdid)
			rdi_ptr.rdiRPOff_ptr.rld_P_esdid = int32(_exi[_rli[i].rli_in_exi_ix].exi_esdid)
			rdi_ptr.rdiRPOff_ptr.rld_offset4 = _rli[i].rli_offset
			break

		case SAME_PID:
			rdi_ptr.rdiROff_ptr = new(goff_rld_data_item_ROff)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiROff_ptr))
			rdi_ptr.rld_type = SAME_PID
			rdi_ptr.rdiROff_ptr.rld_R_esdid = int32(_exi[_rli[i].rli_ref_exi_ix].exi_esdid)
			rdi_ptr.rld_data.rld_data_item_byte0 = 1 << 6 // rld_same_P_ID = true
			rdi_ptr.rdiROff_ptr.rld_offset4 = _rli[i].rli_offset
			break

		case SAME_OFFSET | SAME_PID:
			rdi_ptr.rdiR_ptr = new(goff_rld_data_item_R)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiR_ptr))
			rdi_ptr.rld_type = SAME_OFFSET | SAME_PID
			rdi_ptr.rdiR_ptr.rld_R_esdid = int32(_exi[_rli[i].rli_ref_exi_ix].exi_esdid)
			rdi_ptr.rld_data.rld_data_item_byte0 = 0x60 // rld_same_P_ID = rld_same_offset = true 0x01100000
			break

		case SAME_RID:
			rdi_ptr.rdiPOff_ptr = new(goff_rld_data_item_POff)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiPOff_ptr))
			rdi_ptr.rld_type = SAME_RID
			rdi_ptr.rdiPOff_ptr.rld_P_esdid = int32(_exi[_rli[i].rli_in_exi_ix].exi_esdid)
			rdi_ptr.rld_data.rld_data_item_byte0 = 1 << 7 // rld_same_R_ID = true
			rdi_ptr.rdiPOff_ptr.rld_offset4 = _rli[i].rli_offset
			break

		case SAME_OFFSET | SAME_RID:
			rdi_ptr.rdiP_ptr = new(goff_rld_data_item_P)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiP_ptr))
			rdi_ptr.rld_type = SAME_OFFSET | SAME_RID
			rdi_ptr.rdiP_ptr.rld_P_esdid = int32(_exi[_rli[i].rli_in_exi_ix].exi_esdid)
			rdi_ptr.rld_data.rld_data_item_byte0 = 0xa0 // rld_same_R_ID = rld_same_offset = truea 0x1010000
			break

		case SAME_OFFSET:
			rdi_ptr.rdiRP_ptr = new(goff_rld_data_item_RP)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdiRP_ptr))
			rdi_ptr.rld_type = SAME_OFFSET
			rdi_ptr.rdiRP_ptr.rld_P_esdid = int32(_exi[_rli[i].rli_in_exi_ix].exi_esdid)
			rdi_ptr.rdiRP_ptr.rld_R_esdid = int32(_exi[_rli[i].rli_ref_exi_ix].exi_esdid)
			rdi_ptr.rld_data.rld_data_item_byte0 = 1 << 5 // rld_same_offset = true
			break

		case SAME_PID | SAME_RID:
			rdi_ptr.rdinoRP_ptr = new(goff_rld_data_item_no_RP)
			size = int32(unsafe.Sizeof(rdi_ptr.rld_data) + unsafe.Sizeof(*rdi_ptr.rdinoRP_ptr))
			rdi_ptr.rld_type = SAME_PID | SAME_RID
			rdi_ptr.rdinoRP_ptr.rld_offset4 = _rli[i].rli_offset
			rdi_ptr.rld_data.rld_data_item_byte0 = 0xc0 // rld_same_R_ID = rld_same_P_ID = true 0x11000000
			break

		default:
			Diag("Unexpected pattern (%X) seen", current_pattern)
		}

		// Write out the RLD record into the object file with a continuation record.
		writeRLDRecord(rdi_ptr, size)

		previous_pid = current_pid
		previous_rid = current_rid
		previous_offset = current_offset

	}

	return a
}

func writeRLDObjectRecord(record goff_rld_record, data_ptr *goff_rld_data) uint32 {
	a := uint32(0)
	Cput(record.rld_ptv_prefix)
	a += 1
	Cput(record.rld_ptv_flag)
	a += 1
	Cput(record.rld_ptv_version)
	a += 1
	Cput(record.rld_reserved_1)
	a += 1
	Thearch.Wput(uint16(record.rld_data_length))
	a += 2

	if data_ptr.rld_data != nil {
		Cput(data_ptr.rld_data.rld_data_item_byte0)
		a += 1
		Cput(data_ptr.rld_data.rld_data_item_byte1)
		a += 1
		Cput(data_ptr.rld_data.rld_data_item_byte2)
		a += 1
		Cput(data_ptr.rld_data.rld_reserved_3)
		a += 1
		Cput(data_ptr.rld_data.rld_targ_field_byte_len)
		a += 1
		Cput(data_ptr.rld_data.rld_data_item_byte5)
		a += 1
		Cput(data_ptr.rld_data.rld_reserved_5)
		a += 1
		Cput(data_ptr.rld_data.rld_reserved_6)
		a += 1
	}

	switch data_ptr.rld_type {
	case NONE:
		Thearch.Lput(uint32(data_ptr.rdiRPOff_ptr.rld_R_esdid))
		a += 4
		Thearch.Lput(uint32(data_ptr.rdiRPOff_ptr.rld_P_esdid))
		a += 4
		Thearch.Lput(uint32(data_ptr.rdiRPOff_ptr.rld_offset4))
		a += 4
		break

	case SAME_PID:
		Thearch.Lput(uint32(data_ptr.rdiROff_ptr.rld_R_esdid))
		a += 4
		Thearch.Lput(uint32(data_ptr.rdiROff_ptr.rld_offset4))
		a += 4
		break

	case SAME_OFFSET | SAME_PID:
		Thearch.Lput(uint32(data_ptr.rdiR_ptr.rld_R_esdid))
		a += 4
		break

	case SAME_RID:
		Thearch.Lput(uint32(data_ptr.rdiPOff_ptr.rld_P_esdid))
		a += 4
		Thearch.Lput(uint32(data_ptr.rdiPOff_ptr.rld_offset4))
		a += 4
		break

	case SAME_OFFSET | SAME_RID:
		Thearch.Lput(uint32(data_ptr.rdiP_ptr.rld_P_esdid))
		a += 4
		break

	case SAME_OFFSET:
		Thearch.Lput(uint32(data_ptr.rdiRP_ptr.rld_R_esdid))
		a += 4
		Thearch.Lput(uint32(data_ptr.rdiRP_ptr.rld_P_esdid))
		a += 4
		break

	case SAME_PID | SAME_RID:
		Thearch.Lput(uint32(data_ptr.rdinoRP_ptr.rld_offset4))
		a += 4
		break
	}

	i := a % 80
	for i > 0 && i < 80 {
		Cput('0')
		a += 1
		i++
	}
	/*
		        if ctxt.Debugvlog > 1 && ctxt.Bso != nil {
		           fmt.Printf("%02x", record.rld_ptv_prefix)
		           fmt.Printf("%02x", record.rld_ptv_flag)
		           fmt.Printf("%02x", record.rld_ptv_version)
		           fmt.Printf("%02x  ", record.rld_reserved_1)
		           fmt.Printf("%04x  ", record.rld_data_length)

		           if data_ptr.rld_data != nil {
		             fmt.Printf("%02x", data_ptr.rld_data.rld_data_item_byte0)
		             fmt.Printf("%02x", data_ptr.rld_data.rld_data_item_byte1)
		             fmt.Printf("%02x", data_ptr.rld_data.rld_data_item_byte2)
		             fmt.Printf("%02x  ", data_ptr.rld_data.rld_reserved_3)
		             fmt.Printf("%02x", data_ptr.rld_data.rld_targ_field_byte_len)
		             fmt.Printf("%02x", data_ptr.rld_data.rld_data_item_byte5)
		             fmt.Printf("%02x", data_ptr.rld_data.rld_reserved_5)
		             fmt.Printf("%02x\n", data_ptr.rld_data.rld_reserved_6)
		           }
			}
	*/

	return a
}

func writeRLDRecord(data_ptr *goff_rld_data, size int32) {

	var record goff_rld_record
	if size == 0 {
		return
	}

	record.rld_ptv_prefix = GOFF_PTV_PREFIX
	record.rld_ptv_flag = GOFF_RLD << 4 // rld_ptv_type = GOFF_RLD

	if size <= int32(unsafe.Sizeof(record.rld_data)) {
		record.rld_data_length = int16(size)
		record.rld_ptv_flag = 0xFD & record.rld_ptv_flag // rld_ptv_continuation = false;
		record.rld_ptv_flag = 0xFE & record.rld_ptv_flag // rld_ptv_continued = false;
		writeRLDObjectRecord(record, data_ptr)
	} else {
		Diag("rld data size too large - need to write continuation record\n")
	}

}

func goffwriteEnd() uint32 {
	gend.end_ptv_prefix = uint8(GOFF_PTV_PREFIX)
	gend.end_ptv_flag1 = uint8(GOFF_END << 4) // TODO: end_ptv_type

	a := uint32(0)
	Cput(gend.end_ptv_prefix)
	a += 1
	Cput(gend.end_ptv_flag1)
	a += 1
	Cput(gend.end_ptv_version)
	a += 1
	Cput(gend.end_ptv_flag2)
	a += 1
	Cput(gend.end_amode)
	a += 1
	for i := 0; i < 3; i++ {
		Cput(gend.end_reserved_2[i])
		a += 1
	}

	Thearch.Lput(uint32(gend.end_record_count))
	a += 4
	Thearch.Lput(uint32(gend.end_esdid))
	a += 4
	Thearch.Lput(uint32(gend.end_reserved_3))
	a += 4
	Thearch.Lput(uint32(gend.end_offset))
	a += 4
	Thearch.Wput(uint16(gend.end_name_length))
	a += 2

	for i := a % 80; i < 80; i++ {
		Cput('0')
		a += 1
	}
	return a

}

func addDefExiCode(e *exi_entry, name string, parent_exi_ix uint32, esd_type uint32) {
	e.exi_namespace = EXINS_CODE
	e.exf_executable = true
	e.exf_def = true
	e.exi_offset = 0
	e.exi_length = 0
	e.exi_type = esd_type
	e.exi_parent_exi_ix = parent_exi_ix
	e.exi_name = string(encodeSym(name))
	e.exf_xplink = true
	e.exf_mapped = true
	e.exi_esdid = getNextEsdId()
	e.exi_flag = ESDTS_BYTE << 4
}

func addDefExiData(e *exi_entry, name string, parent_exi_ix uint32, esd_type uint32) {
	e.exi_namespace = EXINS_DATA
	e.exi_offset = 0
	e.exi_length = 0
	e.exi_type = esd_type
	e.exi_parent_exi_ix = parent_exi_ix
	e.exi_name = string(encodeSym(name))
	e.exf_xplink = false
	e.exf_mapped = true
	e.exi_esdid = getNextEsdId()
	e.exi_flag = ESDTS_BYTE << 4
}

func addRefExiCode(e *exi_entry, name string, parent_exi_ix uint32, esd_type uint32) {
	e.exi_namespace = EXINS_CODE
	e.exf_executable = true

	e.exi_offset = 0
	e.exi_length = 0
	e.exi_type = esd_type
	e.exi_parent_exi_ix = parent_exi_ix
	e.exi_name = string(encodeSym(name))
	e.exf_xplink = false
	e.exf_mapped = true
	e.exi_esdid = getNextEsdId()
	e.exi_flag = ESDTS_BYTE << 4
}

func addRefExiData(e *exi_entry, name string, parent_exi_ix uint32, esd_type uint32) {

	e.exi_namespace = _exi[parent_exi_ix].exi_namespace
	e.exi_offset = 0
	e.exi_length = 0
	e.exi_type = esd_type
	e.exi_parent_exi_ix = parent_exi_ix
	e.exi_name = string(encodeSym(name))
	e.exf_xplink = false
	e.exf_mapped = true
	e.exi_esdid = getNextEsdId()
	e.exi_flag = ESDTS_BYTE << 4
}

func addTxi(exi_ix uint32, offset int32, length int64, txtData *[]byte, rec_style uint8) {

	// In single SD, the code length is aggregated
	offset = int32(_exi[exi_ix].exi_length)
	_exi[exi_ix].exi_length += uint32(length)

	_cummulativeTXTSize += uint64(length)

	if _cummulativeTXTSize > maxTotalTXTSize {
		Diag("Excess maxTotalTXTSize \n")
	}

	txi_ix := getNextTxiIx()
	te := getNewTxiEntry(txi_ix)
	te.txi_exi_ix = exi_ix
	te.txi_offset = offset
	te.txi_length = uint32(length)
	te.txi_data_ptr = txtData

	te.txi_flag = rec_style << 4 // txi_rec_style = rec_style
	if _exi[exi_ix].exi_first_txi_ix == 0 {
		_exi[exi_ix].exi_first_txi_ix = txi_ix
	} else {
		last_txi_ix := _exi[exi_ix].exi_last_txi_ix
		_txi[last_txi_ix].txi_next_ix = txi_ix
	}
	_exi[exi_ix].exi_last_txi_ix = txi_ix
}

func addRli_T(ref_exi_ix uint32, in_exi_ix uint32, length int32, sign int16, offset int32, rlt_xx int16) {
	rli_ix := getNextRliIx()
	re := getNewRliEntry(rli_ix)
	re.rli_in_exi_ix = in_exi_ix
	re.rli_offset = offset
	re.rli_ref_exi_ix = ref_exi_ix
	re.rli_type = rlt_xx
	re.rli_length = length
	re.rli_sign = sign
}

func addRli(ref_exi_ix uint32, in_exi_ix uint32, length int32, sign int16, offset int32) {
	var rlt_xx int16

	switch _exi[ref_exi_ix].exi_type {
	case EXT_SD:
		Diag("relocation to SD is invalid in GOFF mode")
		// fallthrough is intentional
	case EXT_ED, EXT_ER, EXT_LD:
		rlt_xx = RLT_ADCON
		if sign == RS_NONE {
			rlt_xx = RLT_VCON
		} else if sign == RS_ADA {
			rlt_xx = RLT_ADA
			// chwan - trigger RLT_XVCON
		} else if sign == RS_EP {
			rlt_xx = RLT_XVCON
		}
	case EXT_PR:
		rlt_xx = RLT_QCON
		if !_exi[ref_exi_ix].exf_deferred {
			rlt_xx = RLT_ADCON
		} else if _exi[in_exi_ix].exf_deferred {
			rlt_xx = RLT_ADCON
		}
	default:
		Diag("relocation issue due to invalid type %d", _exi[ref_exi_ix].exi_type)
	}

	addRli_T(ref_exi_ix, in_exi_ix, length, sign, offset, rlt_xx)
}

func dogoff() {
	if !Isgoff {
		return
	}
}

func Asmbgoff(symo int64) {

	buildCELQSTRT()
	buildPPA1()
	buildPPA2()
	if Debug['w'] == 0 { // dwarf enable
		buildPPA4()
	}
	buildCODEPart()
	buildCELQMAIN()
	buildPPA2Chain()
	buildRelocation()

	if Debug['w'] == 0 { // dwarf enable
		buildDebugParts()
	}

	// HDR
	gh := getGoffhdr()

	gh.hdr_ptv_prefix = GOFF_PTV_PREFIX
	gh.hdr_ptv_flag = GOFF_HDR << 4 // .hdr_ptv_type
	gh.hdr_arch_level = 1

	Cseek(0)
	a := int64(0)
	a += int64(goffwriteHdr())

	// ESDs
	a += int64(goffwriteESDsFromExis())

	// TXTs
	a += int64(goffwriteTXTsFromTxis())

	// RLDs
	a += int64(goffwriteRLDsFromRlis())

	// END Card
	a += int64(goffwriteEnd())

	Cflush()
}
