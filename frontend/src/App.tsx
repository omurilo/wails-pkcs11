import React, { useState, useEffect } from 'react';
import './App.css';
import { GetKeyLabelsForSlot, GetSlots, EncryptFile, DecryptFile, SelectFileToShowDialog, SelectSaveFileToShowDialog } from '../wailsjs/go/main/App';

function App() {
  const [slots, setSlots] = useState<Record<string, string>>({});
  const [selectedSlot, setSelectedSlot] = useState<string>('');
  const [pin, setPin] = useState<string>('9876'); // PIN padrão do SoftHSM
  const [message, setMessage] = useState<string>('');
  const [messageType, setMessageType] = useState<'success' | 'error' | 'info'>('info');
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [keyLabels, setKeyLabels] = useState<string[]>([]);
  const [selectedKeyLabel, setSelectedKeyLabel] = useState<string>('');
  const [isKeyLoading, setIsKeyLoading] = useState<boolean>(false);

  useEffect(() => {
    async function fetchSlots() {
      try {
        const availableSlots = await GetSlots();
        setSlots(availableSlots);
        const firstSlotId = Object.keys(availableSlots)[0];
        if (firstSlotId) {
          setSelectedSlot(firstSlotId);
        }
      } catch (error: any) {
        setMessage(error);
        setMessageType('error');
      }
    }
    fetchSlots();
  }, []);

  useEffect(() => {
    async function fetchKeyLabels() {
      if (!selectedSlot || !pin) {
        setKeyLabels([]);
        setSelectedKeyLabel('');
        return;
      }

      setIsKeyLoading(true);
      try {
        const labels = await GetKeyLabelsForSlot(parseInt(selectedSlot, 10), pin);
        setKeyLabels(labels);
        if (labels.length > 0) {
          setSelectedKeyLabel(labels[0]); // Seleciona o primeiro por padrão
        } else {
          setSelectedKeyLabel(''); // Limpa seleção se não houver labels
        }
      } catch (error: any) {
        setMessage(error.toString()); // Mostra erro de PIN incorreto, por exemplo
        setMessageType('error');
        setKeyLabels([]); // Limpa os labels em caso de erro
        setSelectedKeyLabel('');
      } finally {
        setIsKeyLoading(false);
      }
    }

    fetchKeyLabels();
  }, [selectedSlot, pin]);

  const handleEncrypt = async () => {
    if (!selectedSlot || !pin || !selectedKeyLabel) {
      setMessage('Por favor, preencha todos os campos: Slot, PIN e Label da Chave.');
      setMessageType('error');
      return;
    }

    try {
      // CORREÇÃO FINAL: Chamando nosso método de backend para abrir o diálogo
      const inputFile = await SelectFileToShowDialog(
        "Selecione o arquivo com a chave de unseal",
        "", // filterPattern
        "All Files" // filterDisplayName
      );
      if (!inputFile) return; // Usuário cancelou

      // CORREÇÃO FINAL: Chamando nosso método de backend para salvar o arquivo
      const outputFile = await SelectSaveFileToShowDialog(
        "Salvar arquivo criptografado como",
        "unseal-key.enc"
      );
      if (!outputFile) return; // Usuário cancelou

      setIsLoading(true);
      setMessage('Criptografando, aguarde...');
      setMessageType('info');

      const result = await EncryptFile(parseInt(selectedSlot, 10), pin, selectedKeyLabel, inputFile, outputFile);

      setMessage(result);
      setMessageType('success');
    } catch (error: any) {
      setMessage(`Erro ao criptografar: ${error}`);
      setMessageType('error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDecryptAndUnseal = async () => {
    if (!selectedSlot || !pin || !selectedKeyLabel) {
      setMessage('Por favor, preencha todos os campos: Slot, PIN e Label da Chave.');
      setMessageType('error');
      return;
    }

    try {
      // CORREÇÃO FINAL: Chamando nosso método de backend para abrir o diálogo
      const encryptedFile = await SelectFileToShowDialog(
        "Selecione o arquivo criptografado",
        "*.enc", // filterPattern
        "Encrypted Files" // filterDisplayName
      );
      if (!encryptedFile) return; // Usuário cancelou

      setIsLoading(true);
      setMessage('Descriptografando com o token...');
      setMessageType('info');

      const decryptedKey = await DecryptFile(parseInt(selectedSlot, 10), pin, selectedKeyLabel, encryptedFile);

      setMessage('Chave descriptografada! Enviando para o Vault (simulação)...');

      // O scaffold do Vault não precisa de diálogo, então permanece igual
      // const unsealResult = await UnsealVault(decryptedKey);
      // 
      // setMessage(`Resultado da simulação: ${unsealResult}`);
      setMessageType('success');

    } catch (error: any) {
      setMessage(`Erro ao descriptografar: ${error}`);
      setMessageType('error');
    } finally {
      setIsLoading(false);
    }
  };

  // Nenhuma alteração no JSX abaixo
  return (
    <div id="App">
      <div className="container">
        <h1>Vault PKCS#11 Sealer</h1>
        <div className="form-group">
          <label htmlFor="slot-select">Token PKCS#11 Disponível:</label>
          <select id="slot-select" value={selectedSlot} onChange={(e) => setSelectedSlot(e.target.value)} disabled={isLoading}>
            {Object.entries(slots).map(([id, name]) => (
              <option key={id} value={id}>{name}</option>
            ))}
          </select>
        </div>
        <div className="form-group">
          <label htmlFor="pin-input">PIN do Usuário:</label>
          <input id="pin-input" type="password" value={pin} onChange={(e) => setPin(e.target.value)} disabled={isLoading} />
        </div>
        <div className="form-group">
          <label htmlFor="keylabel-select">Label da Chave:</label>
          <select id="keylabel-select" value={selectedKeyLabel} onChange={(e) => setSelectedKeyLabel(e.target.value)} disabled={isLoading || isKeyLoading}>
            {isKeyLoading ? (
              <option>Carregando chaves...</option>
            ) : keyLabels.length > 0 ? (
              keyLabels.map((label) => (
                <option key={label} value={label}>{label}</option>
              ))
            ) : (
              <option value="">-- Nenhuma chave encontrada --</option>
            )}
          </select>
        </div>

        <div className="button-group">
          <button onClick={handleEncrypt} disabled={isLoading}>Criptografar Chave</button>
          <button onClick={handleDecryptAndUnseal} disabled={isLoading}>Descriptografar e Fazer Unseal (Simulado)</button>
        </div>

        {message && (
          <div className={`message ${messageType}`}>
            {isLoading ? <div className="spinner"></div> : null}
            <p>{message}</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
