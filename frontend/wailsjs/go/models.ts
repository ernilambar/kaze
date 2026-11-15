export namespace main {
	
	export class Document {
	    path: string;
	    name: string;
	    kind: string;
	    content: string;
	    src: string;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.content = source["content"];
	        this.src = source["src"];
	    }
	}
	export class EditorSettings {
	    font_family: string;
	    font_size: number;
	    font_weight: number;
	
	    static createFrom(source: any = {}) {
	        return new EditorSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.font_family = source["font_family"];
	        this.font_size = source["font_size"];
	        this.font_weight = source["font_weight"];
	    }
	}
	export class FolderEntry {
	    path: string;
	    name: string;
	    is_dir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FolderEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.is_dir = source["is_dir"];
	    }
	}
	export class LaunchRequest {
	    files: string[];
	    folders: string[];
	    missing: string[];
	
	    static createFrom(source: any = {}) {
	        return new LaunchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.files = source["files"];
	        this.folders = source["folders"];
	        this.missing = source["missing"];
	    }
	}
	export class SidebarSettings {
	    folders: string[];
	
	    static createFrom(source: any = {}) {
	        return new SidebarSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folders = source["folders"];
	    }
	}
	export class Settings {
	    editor: EditorSettings;
	    sidebar: SidebarSettings;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.editor = this.convertValues(source["editor"], EditorSettings);
	        this.sidebar = this.convertValues(source["sidebar"], SidebarSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

