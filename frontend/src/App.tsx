import React, { useState, useEffect } from 'react';
import { Shield, Key, Usb, RefreshCw, AlertTriangle, CheckCircle, Clock, FileLock, Unlock, Settings, X, FolderOpen } from 'lucide-react';

// Importando as funções do Go
import {
  SaveConfig,
  GetConfig,
  SelectModuleFile,
  GetSlots,
  GetKeyLabelsForSlot,
  EncryptFile,
  DecryptFile,
  UnsealVault,
  SelectFileToShowDialog,
  SelectSaveFileToShowDialog
} from '../wailsjs/go/main/App';
import type { backend } from '../wailsjs/go/models'

interface DeviceCardProps {
  slotId: number;
  slotName: string;
  onEncrypt: (slotId: number, pin: string, keyLabel: string) => void;
  onDecrypt: (slotId: number, pin: string, keyLabel: string) => void;
}

const DeviceCard: React.FC<DeviceCardProps> = ({ slotId, slotName, onEncrypt, onDecrypt }) => {
  const [pin, setPin] = useState<string>('9876');
  const [keyLabels, setKeyLabels] = useState<string[]>([]);
  const [selectedKeyLabel, setSelectedKeyLabel] = useState<string>('');
  const [isKeyLoading, setIsKeyLoading] = useState<boolean>(false);
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [error, setError] = useState<string>('');

  useEffect(() => {
    if (!isExpanded || pin.length < 4) {
      setKeyLabels([]);
      setSelectedKeyLabel('');
      setError(''); // Limpa erro ao recolher ou PIN inválido
      return;
    };

    const fetchKeyLabels = async () => {
      setIsKeyLoading(true);
      setError('');
      try {
        const labels = await GetKeyLabelsForSlot(slotId, pin);
        setKeyLabels(labels);
        if (labels.length > 0) {
          setSelectedKeyLabel(labels[0]);
        } else {
          setSelectedKeyLabel('');
        }
      } catch (err: any) {
        // Aumenta a clareza do erro de PIN para o usuário
        if (err.toString().includes("falha no login")) {
          setError("Invalid PIN for this token.");
        } else {
          setError(err.toString());
        }
        setKeyLabels([]);
        setSelectedKeyLabel('');
      } finally {
        setIsKeyLoading(false);
      }
    };

    const debounceTimer = setTimeout(fetchKeyLabels, 500); // Aumentado para 500ms
    return () => clearTimeout(debounceTimer);

  }, [pin, isExpanded, slotId]);

  const handleEncryptClick = () => {
    onEncrypt(slotId, pin, selectedKeyLabel);
  };

  const handleDecryptClick = () => {
    onDecrypt(slotId, pin, selectedKeyLabel);
  };

  return (
    <div className="border border-secondary-200 bg-secondary-50 rounded-lg p-4 shadow-md transition-all duration-300">
      <div className="flex items-start justify-between">
        <div className="flex items-center space-x-3">
          <div className="w-3 h-3 rounded-full bg-green-500 mt-1 flex-shrink-0" />
          <div>
            <h3 className="font-medium text-secondary-900">{slotName.split('(')[0]}</h3>
            <p className="text-sm text-secondary-600">{slotName.match(/\((.*?)\)/)?.[1]}</p>
            <p className="text-xs text-secondary-500">Slot ID: {slotId}</p>
          </div>
        </div>
        {!isExpanded && (
          <button
            onClick={() => setIsExpanded(true)}
            className="text-sm bg-primary-500 text-white px-3 py-1 rounded-md hover:bg-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-400"
          >
            Authenticate
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="mt-4 pt-4 border-t border-secondary-200 space-y-3">
          <div>
            <label className="text-sm font-medium text-secondary-700 block mb-1">PIN</label>
            <input
              type="password"
              placeholder="Enter PIN"
              value={pin}
              onChange={(e) => setPin(e.target.value)}
              className="w-full text-sm px-2 py-1.5 border border-secondary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-400 bg-white text-secondary-900"
            />
          </div>
          <div>
            <label className="text-sm font-medium text-secondary-700 block mb-1">Key Label</label>
            <select
              value={selectedKeyLabel}
              onChange={(e) => setSelectedKeyLabel(e.target.value)}
              disabled={isKeyLoading || keyLabels.length === 0}
              className="w-full text-sm px-2 py-1.5 border border-secondary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-400 disabled:bg-secondary-100 bg-white text-secondary-900"
            >
              {isKeyLoading ? (
                <option>Loading keys...</option>
              ) : keyLabels.length > 0 ? (
                keyLabels.map((label) => <option key={label} value={label}>{label}</option>)
              ) : (
                <option value="">No keys found for this PIN</option>
              )}
            </select>
          </div>
          {error && <p className="text-xs text-red-600 mt-1">{error}</p>}

          <div className="flex space-x-2 pt-2">
            <button
              onClick={handleEncryptClick}
              disabled={!selectedKeyLabel || isKeyLoading}
              className="flex-1 flex items-center justify-center space-x-2 text-sm bg-primary-600 text-white px-3 py-2 rounded-md hover:bg-primary-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-primary-400"
            >
              <FileLock className="w-4 h-4" />
              <span>Encrypt</span>
            </button>
            <button
              onClick={handleDecryptClick}
              disabled={!selectedKeyLabel || isKeyLoading}
              className="flex-1 flex items-center justify-center space-x-2 text-sm bg-green-600 text-white px-3 py-2 rounded-md hover:bg-green-700 disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-green-400"
            >
              <Unlock className="w-4 h-4" />
              <span>Decrypt & Unseal</span>
            </button>
          </div>
          <button onClick={() => setIsExpanded(false)} className="text-xs text-primary-500 hover:text-primary-700 hover:underline w-full text-center mt-3">
            Collapse
          </button>
        </div>
      )}
    </div>
  );
};

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: () => void; // Para avisar a app principal que precisa recarregar
}

const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose, onSave }) => {
  const [config, setConfig] = useState<backend.Config | null>(null);

  useEffect(() => {
    if (isOpen) {
      GetConfig().then(setConfig);
    }
  }, [isOpen]);

  const handleSelectFile = async () => {
    const path = await SelectModuleFile();
    if (path && config) {
      setConfig({ ...config, pkcs11_module_path: path });
    }
  };

  const handleSave = async () => {
    if (config) {
      try {
        await SaveConfig(config);
        onSave();
        onClose();
      } catch (err) {
        // Idealmente, mostrar um erro dentro do modal
        alert(`Failed to save config: ${err}`);
      }
    }
  };

  if (!isOpen || !config) return null;

  return (
    <div className="fixed inset-0 bg-secondary-950 bg-opacity-80 flex items-center justify-center z-50">
      <div className="bg-secondary-900 text-secondary-50 p-6 rounded-lg shadow-xl border border-primary-700 w-full max-w-lg">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold flex items-center space-x-2"><Settings className="text-primary-300" /> <span>Configurações</span></h2>
          <button onClick={onClose} className="p-1 rounded-full hover:bg-secondary-700"><X /></button>
        </div>
        <div className="space-y-4">
          <div>
            <label htmlFor="modulePath" className="block text-sm font-medium text-secondary-200 mb-1">Caminho do Módulo PKCS#11 (.so, .dll)</label>
            <div className="flex space-x-2">
              <input
                id="modulePath"
                type="text"
                value={config.pkcs11_module_path}
                onChange={(e) => setConfig({ ...config, pkcs11_module_path: e.target.value })}
                placeholder="Ex: /usr/lib/libsofthsm2.so"
                className="flex-grow bg-secondary-800 border border-secondary-600 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-400"
              />
              <button onClick={handleSelectFile} className="bg-primary-700 hover:bg-primary-600 px-4 rounded-md flex items-center space-x-2"><FolderOpen className="w-4 h-4" /> <span>Procurar</span></button>
            </div>
            <p className="text-xs text-secondary-400 mt-2">Se deixado em branco, a aplicação tentará detectar automaticamente na próxima inicialização.</p>
          </div>
          <div className="flex justify-end space-x-3 pt-4">
            <button onClick={onClose} className="bg-secondary-700 hover:bg-secondary-600 text-white px-4 py-2 rounded-md">Cancelar</button>
            <button onClick={handleSave} className="bg-primary-500 hover:bg-primary-600 text-white px-4 py-2 rounded-md">Salvar e Reinicializar</button>
          </div>
        </div>
      </div>
    </div>
  );
};


// --- COMPONENTE PRINCIPAL DA APLICAÇÃO ---
const VaultSealerApp: React.FC = () => {
  const [slots, setSlots] = useState<Record<string, string>>({});
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isDetecting, setIsDetecting] = useState<boolean>(false);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState<string>('');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false); // Estado para o modal

  const detectDevices = async () => {
    setIsDetecting(true);
    setError('');
    setSuccess('');
    try {
      const availableSlots = await GetSlots();
      setSlots(availableSlots);
    } catch (err: any) {
      setError(`Failed to detect devices: ${err}`);
    } finally {
      setIsDetecting(false);
    }
  };

  useEffect(() => {
    detectDevices();
  }, []);

  const handleGlobalAction = async (action: Promise<any>, successMessage: string, errorMessage: string) => {
    setIsLoading(true);
    setError('');
    setSuccess('');
    try {
      const result = await action;
      setSuccess(`${successMessage}: ${result}`);
    } catch (err: any) {
      setError(`${errorMessage}: ${err}`);
    } finally {
      setIsLoading(false);
    }
  };

  const handleEncrypt = async (slotId: number, pin: string, keyLabel: string) => {
    try {
      const inputFile = await SelectFileToShowDialog("Select the unseal key file", "*.*", "All Files");
      if (!inputFile) return;

      const outputFile = await SelectSaveFileToShowDialog("Save encrypted file as", "unseal-key.enc");
      if (!outputFile) return;

      await handleGlobalAction(
        EncryptFile(slotId, pin, keyLabel, inputFile, outputFile),
        'File encrypted successfully',
        'File encryption failed'
      );
    } catch (err: any) {
      // Se o usuário cancelar o diálogo de arquivo, não exibe erro.
      if (!err.toString().includes("cancelled")) {
        setError(`File dialog error: ${err}`);
      }
    }
  };

  const handleDecrypt = async (slotId: number, pin: string, keyLabel: string) => {
    try {
      const encryptedFile = await SelectFileToShowDialog("Select the encrypted file", "*.enc", "Encrypted Files");
      if (!encryptedFile) return;

      setIsLoading(true);
      setError('');
      setSuccess('');
      try {
        const decryptedKey = await DecryptFile(slotId, pin, keyLabel, encryptedFile);
        setSuccess('Key decrypted! Submitting to Vault (simulation)...');

        // Chamar o scaffold do Vault
        const unsealResult = await UnsealVault(decryptedKey);
        setSuccess(`Simulation result: ${unsealResult}`);
      } catch (err: any) {
        setError(`Decryption/Unseal failed: ${err}`);
      } finally {
        setIsLoading(false);
      }

    } catch (err: any) {
      // Se o usuário cancelar o diálogo de arquivo, não exibe erro.
      if (!err.toString().includes("cancelled")) {
        setError(`File dialog error: ${err}`);
      }
    }
  };


  return (
    <div className="min-h-screen bg-secondary-950 p-6 font-sans text-secondary-100">
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        onSave={detectDevices} // Após salvar, chama detectDevices para recarregar
      />

      <div className="max-w-2xl mx-auto">
        <div className="bg-primary-900 rounded-lg shadow-xl p-6 mb-6 border border-primary-700 flex justify-between items-center">
          <div className="flex items-center space-x-4">
            <Shield className="w-10 h-10 text-primary-300" />
            <div>
              <h1 className="text-3xl font-bold text-primary-50">Vault PKCS#11 Sealer</h1>
              <p className="text-sm text-primary-200 mt-1">Securely encrypt and decrypt Vault unseal keys.</p>
            </div>
          </div>
          <button onClick={() => setIsSettingsOpen(true)} className="p-2 rounded-full hover:bg-primary-700 transition-colors">
            <Settings className="text-primary-200" />
          </button>
        </div>
        {/* Mensagens de Erro ou Sucesso */}
        {error && (
          <div className="bg-red-900 bg-opacity-70 border border-red-700 rounded-lg p-4 mb-6 flex items-start space-x-3 text-red-100">
            <AlertTriangle className="w-5 h-5 text-red-400 mt-0.5 flex-shrink-0" />
            <div>
              <h3 className="font-medium text-red-50">Error</h3>
              <p className="text-red-200 text-sm mt-1">{error}</p>
              <button onClick={() => setError('')} className="text-red-300 hover:text-red-100 text-sm mt-2 underline">Dismiss</button>
            </div>
          </div>
        )}
        {success && (
          <div className="bg-green-900 bg-opacity-70 border border-green-700 rounded-lg p-4 mb-6 flex items-start space-x-3 text-green-100">
            <CheckCircle className="w-5 h-5 text-green-400 mt-0.5 flex-shrink-0" />
            <div>
              <h3 className="font-medium text-green-50">Success</h3>
              <p className="text-green-200 text-sm mt-1">{success}</p>
              <button onClick={() => setSuccess('')} className="text-green-300 hover:text-green-100 text-sm mt-2 underline">Dismiss</button>
            </div>
          </div>
        )}


        {/* Seção de Detecção de Dispositivos */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-secondary-50 flex items-center space-x-2">
              <Usb className="w-5 h-5 text-primary-300" />
              <span>PKCS#11 Devices Detected</span>
            </h2>
            <button
              onClick={detectDevices}
              disabled={isDetecting}
              className="flex items-center space-x-2 px-3 py-1.5 text-sm bg-primary-800 text-primary-100 hover:bg-primary-700 rounded-lg disabled:opacity-50 border border-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-400"
            >
              <RefreshCw className={`w-4 h-4 ${isDetecting ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>

          {Object.keys(slots).length === 0 && (
            <div className="text-center py-12 text-secondary-400 bg-secondary-900 border border-secondary-700 rounded-lg shadow-inner">
              <Usb className="w-12 h-12 mx-auto mb-3 text-secondary-600" />
              <p className="font-semibold">Nenhum dispositivo detectado.</p>
              {error && error.includes("não está configurado") ? (
                <p className="text-sm mt-2">
                  O módulo PKCS#11 não foi encontrado. <br />
                  <button onClick={() => setIsSettingsOpen(true)} className="text-primary-400 hover:underline font-semibold">
                    Clique aqui para configurar o caminho da biblioteca.
                  </button>
                </p>
              ) : error ? (
                <p className="text-sm mt-1 text-red-400">{error}</p>
              ) : (
                <p className="text-sm mt-1">Conecte seu token de hardware ou verifique sua configuração.</p>
              )}
            </div>
          )}
          {Object.keys(slots).length > 0 && (
            <div className="space-y-4">
              {Object.entries(slots).map(([id, name]) => (
                <DeviceCard
                  key={id}
                  slotId={parseInt(id, 10)}
                  slotName={name}
                  onEncrypt={handleEncrypt}
                  onDecrypt={handleDecrypt}
                />
              ))}
            </div>
          )}        </div>
        {isLoading && (
          <div className="fixed inset-0 bg-secondary-950 bg-opacity-70 flex items-center justify-center z-50">
            <div className="flex items-center space-x-3 bg-secondary-800 text-secondary-50 p-4 rounded-lg shadow-xl border border-primary-600">
              <Clock className="w-5 h-5 animate-spin text-primary-300" />
              <span className="text-primary-100">Processing with hardware token...</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default VaultSealerApp;
