import tcp from "k6/x/tcp";
import { check } from "k6";

export default function () {
    const conn = tcp.connect(":9000");
    tcp.writeLn(conn, "Say Hello");
    let res = String.fromCharCode(...tcp.read(conn, 1024));
    check(res, {
        "verify ag tag": (res) => res.includes("Hello"),
    });
    tcp.close(conn);
}
