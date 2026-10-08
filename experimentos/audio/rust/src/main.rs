use libloading::{Library, Symbol};
use std::ffi::CString;
use std::os::raw::{c_char, c_int, c_void};
use std::{thread, time::Duration};

fn main() {
    unsafe {
        let sdl = Library::new("libSDL2-2.0.so.0").expect("sdl");
        let mix = Library::new("libSDL2_mixer-2.0.so.0").expect("mixer");
        let sdl_init: Symbol<unsafe extern "C" fn(u32) -> c_int> = sdl.get(b"SDL_Init").unwrap();
        let rw: Symbol<unsafe extern "C" fn(*const c_char, *const c_char) -> *mut c_void> = sdl.get(b"SDL_RWFromFile").unwrap();
        let open: Symbol<unsafe extern "C" fn(c_int, u16, c_int, c_int) -> c_int> = mix.get(b"Mix_OpenAudio").unwrap();
        let load: Symbol<unsafe extern "C" fn(*mut c_void, c_int) -> *mut c_void> = mix.get(b"Mix_LoadWAV_RW").unwrap();
        let play: Symbol<unsafe extern "C" fn(c_int, *mut c_void, c_int, c_int) -> c_int> = mix.get(b"Mix_PlayChannelTimed").unwrap();
        let pos: Symbol<unsafe extern "C" fn(c_int, i16, u8) -> c_int> = mix.get(b"Mix_SetPosition").unwrap();
        sdl_init(0x10);
        assert_eq!(open(44100, 0x8010, 2, 512), 0);
        let path = CString::new(std::env::args().nth(1).unwrap()).unwrap();
        let chunk = load(rw(path.as_ptr(), b"rb\0".as_ptr() as _), 1);
        for ang in [270i16, 0, 90] {
            let ch = play(-1, chunk, 0, -1);
            pos(ch, ang, 0);
            thread::sleep(Duration::from_millis(700));
        }
        println!("ok");
    }
}
