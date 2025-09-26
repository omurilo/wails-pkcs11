package backend

import (
	"fmt"
	"github.com/miekg/pkcs11"
)

type Pkcs11Handler struct {
	ModulePath string
	Ctx        *pkcs11.Ctx
}

func NewPkcs11Handler(modulePath string) *Pkcs11Handler {
	return &Pkcs11Handler{ModulePath: modulePath}
}

func (h *Pkcs11Handler) Initialize() error {
	p := pkcs11.New(h.ModulePath)
	if p == nil {
		return fmt.Errorf("falha ao carregar módulo PKCS#11 em: %s", h.ModulePath)
	}
	if err := p.Initialize(); err != nil {
		return fmt.Errorf("falha ao inicializar módulo: %w", err)
	}
	h.Ctx = p
	return nil
}

func (h *Pkcs11Handler) Finalize() {
	if h.Ctx != nil {
		h.Ctx.Finalize()
	}
}

func (h *Pkcs11Handler) GetSlotsWithInfo() (map[uint]string, error) {
	slots, err := h.Ctx.GetSlotList(true)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar slots: %w", err)
	}

	slotInfoMap := make(map[uint]string)
	for _, slotID := range slots {
		info, err := h.Ctx.GetTokenInfo(slotID)
		if err != nil {
			continue
		}
		slotInfoMap[slotID] = fmt.Sprintf("Slot %d: %s (S/N: %s)", slotID, info.Label, info.SerialNumber)
	}
	return slotInfoMap, nil
}

func (h *Pkcs11Handler) OpenSession(slotID uint) (pkcs11.SessionHandle, error) {
	return h.Ctx.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
}

func (h *Pkcs11Handler) CloseSession(session pkcs11.SessionHandle) {
	h.Ctx.CloseSession(session)
}

func (h *Pkcs11Handler) Login(session pkcs11.SessionHandle, pin string) error {
	return h.Ctx.Login(session, pkcs11.CKU_USER, pin)
}

func (h *Pkcs11Handler) Logout(session pkcs11.SessionHandle) {
	h.Ctx.Logout(session)
}

func (h *Pkcs11Handler) FindKeyPair(session pkcs11.SessionHandle, keyLabel string) (pkcs11.ObjectHandle, pkcs11.ObjectHandle, error) {
	templatePriv := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
	}
	privKey, err := h.findObject(session, templatePriv)
	if err != nil {
		return 0, 0, fmt.Errorf("chave privada com label '%s' não encontrada", keyLabel)
	}

	templatePub := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, keyLabel),
	}
	pubKey, err := h.findObject(session, templatePub)
	if err != nil {
		return 0, 0, fmt.Errorf("chave pública com label '%s' não encontrada", keyLabel)
	}

	return privKey, pubKey, nil
}

func (h *Pkcs11Handler) findObject(session pkcs11.SessionHandle, template []*pkcs11.Attribute) (pkcs11.ObjectHandle, error) {
	if err := h.Ctx.FindObjectsInit(session, template); err != nil {
		return 0, err
	}
	defer h.Ctx.FindObjectsFinal(session)

	obj, _, err := h.Ctx.FindObjects(session, 1)
	if err != nil {
		return 0, err
	}
	if len(obj) == 0 {
		return 0, fmt.Errorf("nenhum objeto encontrado")
	}
	return obj[0], nil
}

func (h *Pkcs11Handler) ListKeyLabels(session pkcs11.SessionHandle) ([]string, error) {
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
	}

	if err := h.Ctx.FindObjectsInit(session, template); err != nil {
		return nil, fmt.Errorf("FindObjectsInit falhou: %w", err)
	}
	defer h.Ctx.FindObjectsFinal(session)

	var labels []string
	foundHandles, _, err := h.Ctx.FindObjects(session, 100) // Busca até 100 chaves
	if err != nil {
		return nil, fmt.Errorf("FindObjects falhou: %w", err)
	}

	for _, handle := range foundHandles {
		attrs, err := h.Ctx.GetAttributeValue(session, handle, []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, nil),
		})
		if err != nil {
			fmt.Printf("Aviso: Falha ao ler atributos da chave handle %d: %v\n", handle, err)
			continue
		}
		labels = append(labels, string(attrs[0].Value))
	}

	return labels, nil
}
