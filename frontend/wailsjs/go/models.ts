export namespace backend {
	
	export class Config {
	    pkcs11_module_path: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pkcs11_module_path = source["pkcs11_module_path"];
	    }
	}

}

