function stopallmusic()
{
   music111.stop();
   music222.stop();
   music333.stop();
   music444.stop();
   music555.stop();
}
function playsound(sounds)
{
   if(_root.savedata2.data.soundON)
   {
      _root.soundnumber += 1;
      if(_root.soundnumber >= 500)
      {
         _root.soundnumber = 400;
      }
      asdfsound = _root.createEmptyMovieClip("sound" + soundnumber,soundnumber);
      qwersound = new Sound(asdfsound);
      qwersound.attachSound(sounds);
      qwersound.setVolume(50);
      qwersound.start(0,0);
   }
}
function playsound2(sounds)
{
   if(_root.savedata2.data.soundON)
   {
      _root.soundnumber += 1;
      if(_root.soundnumber >= 500)
      {
         _root.soundnumber = 400;
      }
      asdfsound = _root.createEmptyMovieClip("sound" + soundnumber,soundnumber);
      qwersound = new Sound(asdfsound);
      qwersound.attachSound(sounds);
      qwersound.setVolume(100);
      qwersound.start(0,0);
   }
}
function CP(cpn, cpx, cpy, cpr, asdf, asdf2)
{
   pDepth += 1;
   if(pDepth >= 9900)
   {
      pDepth = 1001;
   }
   newMC = _root.attachMovie(cpn,"particle" + pDepth,pDepth);
   newMC._x = cpx;
   newMC._y = cpy;
   newMC._rotation = cpr;
   newMC.asdf = asdf;
   newMC.asdf2 = asdf2;
}
function CP2(cpn, cpx, cpy, cpr, asdf)
{
   pDepth2 += 1;
   if(pDepth2 >= -1001)
   {
      pDepth2 = -2000;
   }
   newMC = _root.attachMovie(cpn,"particle" + pDepth2,pDepth2);
   newMC._x = cpx;
   newMC._y = cpy;
   newMC._rotation = cpr;
   newMC.asdf = asdf;
}
function dropsound()
{
   switch(random(3))
   {
      case 0:
         _root.playsound2("drop1.wav");
         break;
      case 1:
         _root.playsound2("drop2.wav");
         break;
      case 2:
         _root.playsound2("drop3.wav");
      default:
         return;
   }
}
function explodesound()
{
   switch(random(4))
   {
      case 0:
         _root.playsound2("explosion1.wav");
         break;
      case 1:
         _root.playsound2("explosion2.wav");
         break;
      case 2:
         _root.playsound2("explosion3.wav");
         break;
      case 3:
         _root.playsound2("explosion4.wav");
      default:
         return;
   }
}
function hitsound()
{
   switch(random(2))
   {
      case 0:
         _root.playsound2("hit1.wav");
         break;
      case 1:
         _root.playsound2("hit2.wav");
      default:
         return;
   }
}
function diesound()
{
   switch(random(4))
   {
      case 0:
         _root.playsound2("die1.wav");
         break;
      case 1:
         _root.playsound2("die2.wav");
         break;
      case 2:
         _root.playsound2("die3.wav");
         break;
      case 3:
         _root.playsound2("die4.wav");
      default:
         return;
   }
}
function gunsound(wepnum)
{
   switch(wepnum)
   {
      case 1:
         _root.playsound("pistol3.wav");
         break;
      case 2:
         _root.playsound("pistol3.wav");
         break;
      case 3:
         _root.playsound("pistol1.wav");
         break;
      case 4:
         _root.playsound("pistol3.wav");
         break;
      case 5:
         _root.playsound("pistol2.wav");
         break;
      case 6:
         _root.playsound("pistol0.wav");
         break;
      case 8:
         _root.playsound("pistol3.wav");
         break;
      case 9:
         _root.playsound("pistol0.wav");
         break;
      case 10:
         _root.playsound("rifle6.wav");
         break;
      case 11:
         _root.playsound("snipe1.wav");
         break;
      case 12:
         _root.playsound("smg1.wav");
         break;
      case 13:
         _root.playsound("shotgun1.wav");
         break;
      case 14:
         _root.playsound("snipe2.wav");
         break;
      case 15:
         _root.playsound("shotgun3.wav");
         break;
      case 16:
         _root.playsound("snipe3.wav");
         break;
      case 17:
         _root.playsound("rifle2.wav");
         break;
      case 18:
         _root.playsound("shotgun3.wav");
         break;
      case 19:
         _root.playsound("silenced2.wav");
         break;
      case 20:
         _root.playsound("rifle6.wav");
         break;
      case 21:
         _root.playsound("smg3.wav");
         break;
      case 22:
         _root.playsound("smg4.wav");
         break;
      case 23:
         _root.playsound("smg1.wav");
         break;
      case 24:
         _root.playsound("smg2.wav");
         break;
      case 25:
         _root.playsound("smg3.wav");
         break;
      case 26:
         _root.playsound("smg4.wav");
         break;
      case 27:
         _root.playsound("smg1.wav");
         break;
      case 28:
         _root.playsound("smg2.wav");
         break;
      case 29:
         _root.playsound("smg3.wav");
         break;
      case 30:
         _root.playsound("smg4.wav");
         break;
      case 31:
         _root.playsound("smg1.wav");
         break;
      case 32:
         _root.playsound("smg2.wav");
         break;
      case 33:
         _root.playsound("snipe4.wav");
         break;
      case 34:
         _root.playsound("snipe5.wav");
         break;
      case 35:
         _root.playsound("snipe6.wav");
         break;
      case 36:
         _root.playsound("snipe1.wav");
         break;
      case 37:
         _root.playsound("snipe2.wav");
         break;
      case 38:
         _root.playsound("snipe3.wav");
         break;
      case 39:
         _root.playsound("snipe4.wav");
         break;
      case 40:
         _root.playsound("silenced1.wav");
         break;
      case 41:
         _root.playsound("snipe6.wav");
         break;
      case 42:
         _root.playsound("snipe1.wav");
         break;
      case 43:
         _root.playsound("snipe2.wav");
         break;
      case 44:
         _root.playsound("shotgun3.wav");
         break;
      case 45:
         _root.playsound("shotgun2.wav");
         break;
      case 46:
         _root.playsound("shotgun3.wav");
         break;
      case 47:
         _root.playsound("shotgun3.wav");
         break;
      case 48:
         _root.playsound("shotgun1.wav");
         break;
      case 49:
         _root.playsound("shotgun2.wav");
         break;
      case 50:
         _root.playsound("shotgun1.wav");
         break;
      case 51:
         _root.playsound("shotgun3.wav");
         break;
      case 52:
         _root.playsound("shotgun3.wav");
         break;
      case 53:
         _root.playsound("shotgun2.wav");
         break;
      case 54:
         _root.playsound("shotgun3.wav");
         break;
      case 55:
         _root.playsound("rifle3.wav");
         break;
      case 56:
         _root.playsound("rifle6.wav");
         break;
      case 57:
         _root.playsound("rifle1.wav");
         break;
      case 58:
         _root.playsound("silenced2.wav");
         break;
      case 59:
         _root.playsound("rifle3.wav");
         break;
      case 60:
         _root.playsound("rifle4.wav");
         break;
      case 61:
         _root.playsound("rifle5.wav");
         break;
      case 62:
         _root.playsound("rifle1.wav");
         break;
      case 63:
         _root.playsound("rifle2.wav");
         break;
      case 64:
         _root.playsound("rifle3.wav");
         break;
      case 65:
         _root.playsound("lmg.wav");
         break;
      case 66:
         _root.playsound("rifle5.wav");
      default:
         return;
   }
}
p1depth = 10001;
p2depth = 10002;
p3depth = 10003;
p4depth = 10004;
arrow1depth = 10005;
arrow2depth = 10006;
arrow3depth = 10007;
arrow4depth = 10008;
mapfxdepth = 10009;
huddepth = 10010;
teamwindepth = 10011;
pausedepth = 10199;
fadedepth = 10200;
mainmenudepth = 10100;
pDepth = 1001;
pDepth2 = -2000;
gototest = false;
slideprevx = 0;
soundnumber = 400;
_root.showunlocks = 0;
savedata3 = SharedObject.getLocal("arenagamedata3");
if(!savedata3.data.filled)
{
   savedata3.data.filled = true;
   savedata3.data.levelarray = new Array();
   i = 0;
   while(i < 11)
   {
      savedata3.data.levelarray[i] = 0;
      i++;
   }
   savedata3.data.levelarray[0] = 1;
   savedata3.data.levelarray[1] = 1;
   savedata3.data.levelarray[2] = 0;
   savedata3.data.levelarray[3] = 0;
   savedata3.data.levelarray[4] = 0;
   savedata3.data.levelarray[5] = 0;
   savedata3.data.levelarray[6] = 0;
   savedata3.data.levelarray[7] = 0;
   savedata3.data.levelarray[8] = 0;
   savedata3.data.levelarray[9] = 0;
   savedata3.data.p1name = "Player 1";
   savedata3.data.p1color = 2;
   savedata3.data.p1shirt = 1;
   savedata3.data.p1hat = 1;
   savedata3.data.p1gun = 1;
   savedata3.data.p1perk = 7;
   savedata3.data.p1ptype = 1;
   savedata3.data.p2name = "Player 2";
   savedata3.data.p2color = 5;
   savedata3.data.p2shirt = 1;
   savedata3.data.p2hat = 1;
   savedata3.data.p2gun = 1;
   savedata3.data.p2perk = 7;
   savedata3.data.p2ptype = 0;
   savedata3.data.gunarray = new Array();
   i = 0;
   while(i < 57)
   {
      savedata3.data.gunarray[i] = true;
      i++;
   }
   savedata3.data.gunarray[18] = false;
   savedata3.data.gunarray[19] = false;
   savedata3.data.gunarray[20] = false;
   savedata3.data.gunarray[21] = false;
   savedata3.data.gunarray[22] = false;
   savedata3.data.gunarray[29] = false;
   savedata3.data.gunarray[30] = false;
   savedata3.data.gunarray[31] = false;
   savedata3.data.gunarray[32] = false;
   savedata3.data.gunarray[33] = false;
   savedata3.data.gunarray[40] = false;
   savedata3.data.gunarray[41] = false;
   savedata3.data.gunarray[42] = false;
   savedata3.data.gunarray[43] = false;
   savedata3.data.gunarray[44] = false;
   savedata3.data.gunarray[52] = false;
   savedata3.data.gunarray[53] = false;
   savedata3.data.gunarray[54] = false;
   savedata3.data.gunarray[55] = false;
   savedata3.data.gunarray[56] = false;
}
savedata2 = SharedObject.getLocal("arenagamedata2");
if(!savedata2.data.filled)
{
   savedata2.data.filled = true;
   savedata2.data.musicON = true;
   savedata2.data.soundON = true;
   savedata2.data.def_quality = 2;
   savedata2.data.controlarray = new Array();
   i = 0;
   while(i < 4)
   {
      savedata2.data.controlarray[i] = new Array();
      j = 0;
      while(j < 6)
      {
         savedata2.data.controlarray[i][j] = 0;
         j++;
      }
      i++;
   }
   savedata2.data.controlarray[0][0] = 38;
   savedata2.data.controlarray[0][1] = 37;
   savedata2.data.controlarray[0][2] = 40;
   savedata2.data.controlarray[0][3] = 39;
   savedata2.data.controlarray[0][4] = 219;
   savedata2.data.controlarray[0][5] = 221;
   savedata2.data.controlarray[1][0] = 87;
   savedata2.data.controlarray[1][1] = 65;
   savedata2.data.controlarray[1][2] = 83;
   savedata2.data.controlarray[1][3] = 68;
   savedata2.data.controlarray[1][4] = 84;
   savedata2.data.controlarray[1][5] = 89;
   savedata2.data.controlarray[2][0] = 111;
   savedata2.data.controlarray[2][1] = 103;
   savedata2.data.controlarray[2][2] = 104;
   savedata2.data.controlarray[2][3] = 105;
   savedata2.data.controlarray[2][4] = 106;
   savedata2.data.controlarray[2][5] = 109;
   savedata2.data.controlarray[3][0] = 101;
   savedata2.data.controlarray[3][1] = 97;
   savedata2.data.controlarray[3][2] = 98;
   savedata2.data.controlarray[3][3] = 99;
   savedata2.data.controlarray[3][4] = 96;
   savedata2.data.controlarray[3][5] = 110;
}
