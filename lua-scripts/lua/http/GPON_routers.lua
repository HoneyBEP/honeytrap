--
-- Created by IntelliJ IDEA.
-- User: nordi
-- Date: 7-5-2018
-- Time: 15:20
-- To change this template use File | Settings | File Templates.
--

function canHandle()
    return true
end

function handle(message)
    local url = retrieveURLs(message);
    if (url and string.match(url, 'gpon.php')) then
        return "wget: missing URL \nUsage: wget [OPTION]... [URL]... \n\nTry `wget --help' for more options.\n";
    end
    --return message
    return message;
end

function preHandle(message)

end

function afterHandle(message)

end

-- https://stackoverflow.com/questions/23590304/finding-a-url-in-a-string-lua-pattern

-- all characters allowed to be inside URL according to RFC 3986 but without
-- comma, semicolon, apostrophe, equal, brackets and parentheses
-- (as they are used frequently as URL separators)
function retrieveURLs(message)
    local domains = [[.ac.ad.ae.aero.af.ag.ai.al.am.an.ao.aq.ar.arpa.as.asia.at.au
   .aw.ax.az.ba.bb.bd.be.bf.bg.bh.bi.biz.bj.bm.bn.bo.br.bs.bt.bv.bw.by.bz.ca
   .cat.cc.cd.cf.cg.ch.ci.ck.cl.cm.cn.co.com.coop.cr.cs.cu.cv.cx.cy.cz.dd.de
   .dj.dk.dm.do.dz.ec.edu.ee.eg.eh.er.es.et.eu.fi.firm.fj.fk.fm.fo.fr.fx.ga
   .gb.gd.ge.gf.gh.gi.gl.gm.gn.gov.gp.gq.gr.gs.gt.gu.gw.gy.hk.hm.hn.hr.ht.hu
   .id.ie.il.im.in.info.int.io.iq.ir.is.it.je.jm.jo.jobs.jp.ke.kg.kh.ki.km.kn
   .kp.kr.kw.ky.kz.la.lb.lc.li.lk.lr.ls.lt.lu.lv.ly.ma.mc.md.me.mg.mh.mil.mk
   .ml.mm.mn.mo.mobi.mp.mq.mr.ms.mt.mu.museum.mv.mw.mx.my.mz.na.name.nato.nc
   .ne.net.nf.ng.ni.nl.no.nom.np.nr.nt.nu.nz.om.org.pa.pe.pf.pg.ph.pk.pl.pm
   .pn.post.pr.pro.ps.pt.pw.py.qa.re.ro.ru.rw.sa.sb.sc.sd.se.sg.sh.si.sj.sk
   .sl.sm.sn.so.sr.ss.st.store.su.sv.sy.sz.tc.td.tel.tf.tg.th.tj.tk.tl.tm.tn
   .to.tp.tr.travel.tt.tv.tw.tz.ua.ug.uk.um.us.uy.va.vc.ve.vg.vi.vn.vu.web.wf
   .ws.xxx.ye.yt.yu.za.zm.zr.zw]]
    local tlds = {}
    for tld in domains:gmatch'%w+' do
        tlds[tld] = true
    end
    local function max4(a,b,c,d) return math.max(a+0, b+0, c+0, d+0) end
    local protocols = {[''] = 0, ['http://'] = 0, ['https://'] = 0, ['ftp://'] = 0}
    local finished = {}

    for pos_start, url, prot, subd, tld, colon, port, slash, path in
    message:gmatch'()(([%w_.~!*:@&+$/?%%#-]-)(%w[-.%w]*%.)(%w+)(:?)(%d*)(/?)([%w_.~!*:@&+$/?%%#=-]*))'
    do
        if protocols[prot:lower()] == (1 - #slash) * #path and not subd:find'%W%W'
                and (colon == '' or port ~= '' and port + 0 < 65536)
                and (tlds[tld:lower()] or tld:find'^%d+$' and subd:find'^%d+%.%d+%.%d+%.$'
                and max4(tld, subd:match'^(%d+)%.(%d+)%.(%d+)%.$') < 256)
        then
            finished[pos_start] = true
            return url
        end
    end

    for pos_start, url, prot, dom, colon, port, slash, path in
    message:gmatch'()((%f[%w]%a+://)(%w[-.%w]*)(:?)(%d*)(/?)([%w_.~!*:@&+$/?%%#=-]*))'
    do
        if not finished[pos_start] and not (dom..'.'):find'%W%W'
                and protocols[prot:lower()] == (1 - #slash) * #path
                and (colon == '' or port ~= '' and port + 0 < 65536)
        then
            return url
        end
    end

    return nil
end
