function endgame()
{
   newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
   if(!campaignmode)
   {
      newmc.targetframe = 6;
   }
   else
   {
      newmc.targetframe = 4;
   }
   newmc._x = - _root._x;
   newmc._y = - _root._y;
}
function zombielevelup()
{
   zombiewave += 1;
   zombiekilled = zombiewave * 5;
   zombiekilledgoal = zombiewave * 5;
   _root.attachMovie("fx_zombiewaveup","fx_teamwin",_root.teamwindepth);
}
function returntomenu()
{
   if(campaignmode)
   {
      playerwin = true;
      i = 0;
      while(i < activeplayers.length)
      {
         if(activeplayers[i].AI)
         {
            playerwin = false;
            break;
         }
         i++;
      }
      if(playerwin)
      {
         _root.savedata3.data.levelarray[campaignlevel - 1] = 2;
         _root.showunlocks = campaignlevel;
         switch(campaignlevel)
         {
            case 1:
               _root.savedata3.data.gunarray[18] = true;
               _root.savedata3.data.gunarray[29] = true;
               break;
            case 2:
               _root.savedata3.data.gunarray[40] = true;
               _root.savedata3.data.gunarray[52] = true;
               break;
            case 3:
               _root.savedata3.data.gunarray[19] = true;
               _root.savedata3.data.gunarray[30] = true;
               break;
            case 4:
               _root.savedata3.data.gunarray[41] = true;
               _root.savedata3.data.gunarray[53] = true;
               break;
            case 5:
               _root.savedata3.data.gunarray[20] = true;
               _root.savedata3.data.gunarray[31] = true;
               break;
            case 6:
               _root.savedata3.data.gunarray[42] = true;
               _root.savedata3.data.gunarray[54] = true;
               break;
            case 7:
               _root.savedata3.data.gunarray[21] = true;
               _root.savedata3.data.gunarray[32] = true;
               break;
            case 8:
               _root.savedata3.data.gunarray[43] = true;
               _root.savedata3.data.gunarray[55] = true;
               break;
            case 9:
               _root.savedata3.data.gunarray[22] = true;
               _root.savedata3.data.gunarray[33] = true;
               break;
            case 10:
               _root.savedata3.data.gunarray[44] = true;
               _root.savedata3.data.gunarray[56] = true;
         }
      }
   }
   deleteeverything = true;
   deletemc(player1);
   deletemc(player2);
   deletemc(player3);
   deletemc(player4);
   deletemc(ground);
   deletemc(mapscene);
   if(hud)
   {
      deletemc(hud);
   }
   if(arrow1)
   {
      deletemc(arrow1);
   }
   if(arrow2)
   {
      deletemc(arrow2);
   }
   if(arrow3)
   {
      deletemc(arrow3);
   }
   if(arrow4)
   {
      deletemc(arrow4);
   }
   if(mapfx)
   {
      deletemc(mapfx);
   }
   if(game_pause)
   {
      deletemc(game_pause);
   }
   if(mainmenu)
   {
      gotomenu = false;
      removeMovieClip(mainmenu);
      delete mainmenu.onEnterFrame;
   }
   _quality = "HIGH";
}
function deletemc(mc)
{
   mc.swapDepths(1);
   removeMovieClip(mc);
   delete mc.onEnterFrame;
}
stop();
_quality = "HIGH";
switch(savedata2.data.def_quality)
{
   case 1:
      _quality = "LOW";
      break;
   case 2:
      _quality = "MEDIUM";
      break;
   case 3:
      _quality = "HIGH";
}
tournament = false;
GAMEPAUSED = false;
_root.stopallmusic();
if(_root.savedata2.data.musicON)
{
   switch(random(3))
   {
      case 0:
         _root.music333.start(0.1,100);
         break;
      case 1:
         _root.music444.start(0.1,100);
         break;
      case 2:
         _root.music555.start(0.1,100);
   }
}
if(campaignmode)
{
   p3ptype = 0;
   p4ptype = 0;
   gototest = false;
}
if(gotomenu)
{
   attachMovie("mainmenu","mainmenu",mainmenudepth);
   crateON = true;
   powerupON = false;
   gototest = false;
   hud._alpha = 0;
   temp = _root.ground._totalframes;
   do
   {
      _root.mapnumber = random(ground._totalframes) + 1;
   }
   while(mapnumber == temp - 6 || mapnumber == temp - 7 || mapnumber == temp - 8 || mapnumber == 13);
   _root.p1name = "Bot 1";
   _root.p1color = random(10) + 1;
   _root.p1shirt = random(15) + 1;
   _root.p1hat = random(24) + 1;
   _root.p1gun = random(5) + 1;
   _root.p1perk = 0;
   _root.p1ptype = 2;
   _root.p2name = "Bot 2";
   do
   {
      _root.p2color = random(10) + 1;
   }
   while(_root.p2color == _root.p1color);
   _root.p2shirt = random(15) + 1;
   _root.p2hat = random(24) + 1;
   _root.p2gun = random(5) + 1;
   _root.p2perk = 0;
   _root.p2ptype = 2;
   _root.totallives = 1000;
   _root.gamemode = 1;
   _root.p3ptype = 0;
   _root.p4ptype = 0;
   _root.totallives = 10000;
   _root.gamemode = 1;
   _root.campaignmode = false;
   _root.stopallmusic();
   if(_root.savedata2.data.musicON)
   {
      _root.music111.start(0.1,100);
   }
}
if(gototest)
{
   crateON = false;
   powerON = false;
   hud._alpha = 0;
   _root.gamemode = 1;
   _root.totallives = 1000;
   _root.p1name = "Player";
   _root.p1color = 2;
   _root.p1shirt = 1;
   _root.p1hat = 1;
   _root.p1gun = 2;
   if(gototestnumber <= 6)
   {
      _root.p1gun = gototestnumber;
   }
   _root.p1perk = 0;
   _root.p1ptype = 1;
   _root.p2name = "Dummy";
   _root.p2color = 8;
   _root.p2shirt = 24;
   _root.p2hat = 1;
   _root.p2gun = 2;
   _root.p2perk = 0;
   _root.p2ptype = 2;
   _root.campaignmode = false;
}
gravity = 0.88;
friction = 0.93;
airfriction = 0.88;
speed = 0.7;
power = 13.5;
teamgame = false;
if(!teamgame)
{
   p1team = 1;
   p2team = 2;
   p3team = 3;
   p4team = 4;
}
if(gamemode == 5)
{
   _root.totallives = 10;
   crateON = true;
   powerON = false;
   p1team = 0;
   p2team = 0;
   p3team = 0;
   p4team = 0;
}
_root.showunlocks = 0;
if(campaignmode)
{
   powerON = true;
   switch(campaignlevel)
   {
      case 1:
         crateON = false;
         powerON = false;
         _root.totallives = 3;
         _root.mapnumber = 13;
         gamemode = 1;
         _root.p4name = "Dummy";
         _root.p4color = 8;
         _root.p4shirt = 24;
         _root.p4hat = 1;
         _root.p4gun = 0;
         _root.p4perk = 0;
         _root.p4ptype = 2;
         break;
      case 2:
         crateON = true;
         powerON = true;
         _root.totallives = 7;
         _root.mapnumber = 12;
         gamemode = 1;
         _root.p4name = "Caveman Johnson";
         _root.p4color = 8;
         _root.p4shirt = 10;
         _root.p4hat = 7;
         _root.p4gun = 1;
         _root.p4perk = 0;
         _root.p4ptype = 2;
         break;
      case 3:
         _root.totallives = 7;
         _root.mapnumber = 10;
         gamemode = 4;
         _root.p4name = "Santa";
         _root.p4color = 1;
         _root.p4shirt = 11;
         _root.p4hat = 3;
         _root.p4gun = 2;
         _root.p4perk = 7;
         _root.p4ptype = 2;
         break;
      case 4:
         crateON = true;
         _root.totallives = 7;
         _root.mapnumber = 8;
         gamemode = 1;
         powerON = false;
         _root.p4name = "Pylon Man";
         _root.p4color = 9;
         _root.p4shirt = 1;
         _root.p4hat = 9;
         _root.p4gun = 1;
         _root.p4perk = 0;
         _root.p4ptype = 3;
         break;
      case 5:
         crateON = true;
         powerON = true;
         _root.totallives = 7;
         _root.mapnumber = 5;
         gamemode = 1;
         _root.p4name = "Gun Fu Master";
         _root.p4color = 7;
         _root.p4shirt = 9;
         _root.p4hat = 10;
         _root.p4gun = 3;
         _root.p4perk = 6;
         _root.p4ptype = 2;
         break;
      case 6:
         teamgame = true;
         gamemode = 3;
         p1team = 1;
         p2team = 1;
         p3team = 2;
         p4team = 2;
         _root.totallives = 5;
         _root.mapnumber = 11;
         crateON = true;
         powerON = true;
         _root.p3name = "Mafia Guy";
         _root.p3color = 2;
         _root.p3shirt = 3;
         _root.p3hat = 5;
         _root.p3gun = 2;
         _root.p3perk = 0;
         _root.p3ptype = 2;
         _root.p4name = "Mafia Dude";
         _root.p4color = 2;
         _root.p4shirt = 4;
         _root.p4hat = 5;
         _root.p4gun = 3;
         _root.p4perk = 0;
         _root.p4ptype = 2;
         break;
      case 7:
         crateON = true;
         powerON = true;
         _root.totallives = 7;
         _root.mapnumber = 2;
         gamemode = 1;
         _root.p4name = "Jet Pack Bunny";
         _root.p4color = 4;
         _root.p4shirt = 30;
         _root.p4hat = 17;
         _root.p4gun = 4;
         _root.p4perk = 5;
         _root.p4ptype = 4;
         break;
      case 8:
         _root.totallives = 7;
         _root.mapnumber = 3;
         gamemode = 2;
         _root.p4name = "Pirate";
         _root.p4color = 10;
         _root.p4shirt = 8;
         _root.p4hat = 15;
         _root.p4gun = 1;
         _root.p4perk = 5;
         _root.p4ptype = 2;
         break;
      case 9:
         _root.totallives = 7;
         _root.mapnumber = 1;
         gamemode = 1;
         crateON = true;
         powerON = true;
         _root.p4name = "Sinusoidal Sam";
         _root.p4color = 5;
         _root.p4shirt = 5;
         _root.p4hat = 1;
         _root.p4gun = 6;
         _root.p4perk = 2;
         _root.p4ptype = 5;
         break;
      case 10:
         _root.totallives = 5;
         _root.mapnumber = 7;
         gamemode = 1;
         crateON = true;
         powerON = false;
         _root.p4name = "The Boss";
         _root.p4color = 6;
         _root.p4shirt = 14;
         _root.p4hat = 13;
         _root.p4gun = 6;
         _root.p4perk = 6;
         _root.p4ptype = 6;
   }
   if(gamemode == 1 || gamemode == 2)
   {
      if(p2ptype == 1)
      {
         teamgame = true;
      }
   }
   p1team = 1;
   p2team = 1;
}
if(gamemode == 3)
{
   teamgame = true;
}
if(p1ptype == 1)
{
   attachMovie("player","player1",-1);
}
if(p2ptype == 1)
{
   attachMovie("player","player2",-2);
}
if(p3ptype == 1)
{
   attachMovie("player","player3",-3);
}
if(p4ptype == 1)
{
   attachMovie("player","player4",-4);
}
if(p1ptype == 2)
{
   attachMovie("playerAI","player1",-1);
}
if(p2ptype == 2)
{
   attachMovie("playerAI","player2",-2);
}
if(p3ptype == 2)
{
   attachMovie("playerAI","player3",-3);
}
if(p4ptype == 2)
{
   attachMovie("playerAI","player4",-4);
}
if(p4ptype == 3)
{
   attachMovie("playerAI_double","player4",-4);
}
if(p4ptype == 4)
{
   attachMovie("playerAI3","player4",-4);
}
if(p4ptype == 5)
{
   attachMovie("playerAI4","player4",-4);
}
if(p4ptype == 6)
{
   attachMovie("playerAIboss","player4",-4);
}
activeplayers = new Array();
if(player1)
{
   activeplayers[activeplayers.length] = player1;
}
if(player2)
{
   activeplayers[activeplayers.length] = player2;
}
if(player3)
{
   activeplayers[activeplayers.length] = player3;
}
if(player4)
{
   activeplayers[activeplayers.length] = player4;
}
cratearray = new Array();
pgsdata = new Array();
i = 0;
while(i < 4)
{
   pgsdata[i] = new Array();
   j = 0;
   while(j < 10)
   {
      pgsdata[i][j] = 0;
      if(i == 0 && !player1)
      {
         pgsdata[i][j] = -1;
      }
      if(i == 1 && !player2)
      {
         pgsdata[i][j] = -1;
      }
      if(i == 2 && !player3)
      {
         pgsdata[i][j] = -1;
      }
      if(i == 3 && !player4)
      {
         pgsdata[i][j] = -1;
      }
      j++;
   }
   i++;
}
pgsdata[0][0] = p1name;
pgsdata[1][0] = p2name;
pgsdata[2][0] = p3name;
pgsdata[3][0] = p4name;
cratetime = 200;
poweruptime = 0;
zombietime = 0;
zombiewave = 0;
zombiekilled = 0;
zombiekilledtotal = 0;
zombiekilledgoal = 0;
if(gamemode == 5)
{
   zombielevelup();
}
deleteeverything = false;
gamewin = false;
teamgamewin = false;
gamewincountdown = 0;
if(!campaignmode)
{
   campaignlevel = -1;
}
dynamitetime = 0;
this.onEnterFrame = function()
{
   if(!GAMEPAUSED)
   {
      cratetime += 1;
      if(cratetime >= 400 && !gototest && !tournament && crateON)
      {
         if(gamemode != 4)
         {
            _root.CP("crate",_root.ground._x + _root.ground.cratearea._x + random(_root.ground.cratearea._width),-300,0,0);
         }
         cratetime = 0;
      }
      if(!gotomenu && gamemode != 4 && gamemode != 5 && !gototest && !tournament && powerON)
      {
         poweruptime += 1;
         if(poweruptime >= 600)
         {
            do
            {
               powerupx = ground._x + ground.platform._x + 20 + random(ground.platform._width - 40);
               powerupy = ground._y + ground.platform._y + random(ground.platform._height);
            }
            while(!_root.ground.platform.hitTest(powerupx,powerupy,true));
            i = 1;
            while(i < i + 1)
            {
               if(!_root.ground.platform.hitTest(powerupx,powerupy - i * 2,true))
               {
                  powerupy -= i * 2;
                  break;
               }
               i++;
            }
            _root.CP("powerup",powerupx,powerupy - 40,0,0);
            poweruptime = 0;
         }
      }
      if(!gamewin)
      {
         maxleft = activeplayers[0]._x;
         i = 1;
         while(i < activeplayers.length)
         {
            if(activeplayers[i].iszombie)
            {
               break;
            }
            if(activeplayers[i]._x < maxleft)
            {
               maxleft = activeplayers[i]._x;
            }
            if(i == 3)
            {
               break;
            }
            i++;
         }
         maxright = activeplayers[0]._x;
         i = 1;
         while(i < activeplayers.length)
         {
            if(activeplayers[i].iszombie)
            {
               break;
            }
            if(activeplayers[i]._x > maxright)
            {
               maxright = activeplayers[i]._x;
            }
            if(i == 3)
            {
               break;
            }
            i++;
         }
         if(maxleft < -100)
         {
            maxleft = -100;
         }
         if(maxright > 1000)
         {
            maxright = 1000;
         }
         _root._x += ((maxleft + maxright) / -2 + 450 - _root._x) / 8;
         _root._x = Math.round(_root._x);
         maxhigh = activeplayers[0]._y;
         i = 1;
         while(i < activeplayers.length)
         {
            if(activeplayers[i].iszombie)
            {
               break;
            }
            if(activeplayers[i]._y < maxhigh)
            {
               maxhigh = activeplayers[i]._y;
            }
            if(i == 3)
            {
               break;
            }
            i++;
         }
         maxlow = activeplayers[0]._y;
         i = 1;
         while(i < activeplayers.length)
         {
            if(activeplayers[i].iszombie)
            {
               break;
            }
            if(activeplayers[i]._y > maxlow)
            {
               maxlow = activeplayers[i]._y;
            }
            if(i == 3)
            {
               break;
            }
            i++;
         }
         if(maxhigh < 50)
         {
            maxhigh = 50;
         }
         if(maxlow < 50)
         {
            maxlow = 50;
         }
         if(maxlow > 500)
         {
            maxlow = 500;
         }
         if(maxhigh > 500)
         {
            maxhigh = 500;
         }
         _root._y += ((maxhigh + maxlow) / -2 + 280 - _root._y) / 8;
         _root._y = Math.round(_root._y);
         _root.hud._x = - _root._x;
         _root.hud._y = - _root._y;
         _root.mapscene.scene1.update();
         _root.mapscene.scene2.update();
         _root.mainmenu._x = - _root._x;
         _root.mainmenu._y = - _root._y;
      }
      else if(gamewin)
      {
         _root._x = (- activeplayers[0]._x) * (_xscale / 100) + 450;
         _root._y = (- activeplayers[0]._y) * (_xscale / 100) + 400;
         _root._xscale = 200;
         _root._yscale = 200;
         if(hud)
         {
            deletemc(hud);
         }
         if(arrow1)
         {
            deletemc(arrow1);
         }
         if(arrow2)
         {
            deletemc(arrow2);
         }
         if(arrow3)
         {
            deletemc(arrow3);
         }
         if(arrow4)
         {
            deletemc(arrow4);
         }
      }
      if(gamemode == 5)
      {
         zombietime += 1;
         if(zombietime >= 60 - zombiewave * 2 && zombiekilledgoal > 0)
         {
            if(gamemode == 5 && activeplayers.length < 5 + zombiewave)
            {
               zombiekilledgoal -= 1;
               _root.CP("player_zombie",_root.ground._x + _root.ground.cratearea._x + random(_root.ground.cratearea._width),-300,0,0);
               activeplayers[activeplayers.length] = newMC;
            }
            zombietime = 0;
         }
         else if(zombietime >= 60 - zombiewave * 2 && zombiekilled <= 0)
         {
            zombielevelup();
         }
      }
      if(teamgame && !teamgamewin)
      {
         teamwin = _root.activeplayers[0].teamnumber;
         j = 1;
         while(j < _root.activeplayers.length)
         {
            if(_root.activeplayers[j].teamnumber != teamwin)
            {
               teamwin = -1;
               break;
            }
            j++;
         }
         if(teamwin != -1)
         {
            teamgamewin = true;
            newmc = _root.attachMovie("fx_teamwin","fx_teamwin",_root.teamwindepth);
            newmc.asdf = teamwin;
         }
      }
      if(gamewin || teamgamewin)
      {
         gamewincountdown += 1;
         if(gamewincountdown >= 100)
         {
            gamewincountdown = -9999;
            endgame();
         }
      }
      if(Key.isDown(27) && !fadeaway && !gotomenu && !gamewin || Key.isDown(32) && !fadeaway && !gotomenu && !gamewin)
      {
         if(gototest)
         {
            newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
            newmc.targetframe = 8;
            newmc._x = - _root._x;
            newmc._y = - _root._y;
         }
         else
         {
            this.attachMovie("game_pause","game_pause",pausedepth);
         }
      }
      if(Key.isDown(45) && !tournament)
      {
         tournament = true;
      }
      if(campaignlevel == 5)
      {
         dynamitetime += 1;
         if(dynamitetime >= 80)
         {
            dynamitetime = 0;
            _root.CP("wep_grenade",_root.ground._x + _root.ground.cratearea._x + random(_root.ground.cratearea._width),-200,0,-1000);
         }
      }
   }
};
