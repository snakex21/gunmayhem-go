function gotkilled()
{
   i = 0;
   while(i < _root.activeplayers.length)
   {
      if(_root.activeplayers[i] == this)
      {
         _root.activeplayers.splice(i,1);
      }
      i++;
   }
   _root.zombiekilled -= 1;
   _root.zombiekilledtotal += 1;
   this.swapDepths(1);
   removeMovieClip(this);
   delete this.onEnterFrame;
}
function SELFDESTRUCT()
{
   _root.CP("fx_ex6",_X,_Y - 20,0);
   i = 0;
   while(i < 5)
   {
      _root.CP("fx_ex5",_X,_Y - 20,0);
      i++;
   }
   i = 0;
   while(i < 5)
   {
      _root.CP("fx_ex3",_X,_Y - 20,0);
      i++;
   }
   j = 0;
   while(j < 5)
   {
      _root.CP("fx_blood",_X,_Y,0,0);
      j++;
   }
   _root.CP("fx_combo",_X,_Y - 50,0,-5);
   _root.CP("fx_bloodstain",_X,_Y,0,0);
   gotkilled();
}
AI = true;
iszombie = true;
vy = 0;
vx = 0;
jumpnum = 0;
jumpkey = false;
nadekey = false;
downkey = false;
freepass = false;
facing = 1;
walkanim = 0;
offhandnumber = 1;
offhandammo = 3;
gothitby = "none";
hitbynade = false;
greedykill = false;
cheapshottimer = 0;
destructkey = false;
waittime = 100;
idletime = 40;
idlemax = 40;
nadepower = 0;
wepnumber = 1;
rof = 12;
firepower = 25;
recoil = 0.8;
blowback = 30;
bullets = 9999;
idlerotate = 40;
pushback = 0;
disabled = false;
hitnumber = 0;
hittimer = 0;
_xscale = 80;
_yscale = 80;
KEYUP = false;
KEYDOWN = false;
KEYLEFT = false;
KEYRIGHT = false;
KEYSHOOT = false;
KEYNADE = false;
doubletime = -100;
displayname = "zombie";
playernumber = 8;
if(this._name == "double")
{
   doubletime = 600;
   _root.activeplayers[_root.activeplayers.length] = this;
   lives = 1;
   PLAYERNUMBER = asdf;
   switch(asdf)
   {
      case 1:
         playernumber = _root.p1color;
         displayname = _root.p1name;
         defaultgun = _root.p1gun;
         teamnumber = _root.p1team;
         break;
      case 2:
         playernumber = _root.p2color;
         displayname = _root.p2name;
         defaultgun = _root.p2gun;
         teamnumber = _root.p2team;
         break;
      case 3:
         playernumber = _root.p3color;
         displayname = _root.p3name;
         defaultgun = _root.p3gun;
         teamnumber = _root.p3team;
         break;
      case 4:
         playernumber = _root.p4color;
         displayname = _root.p4name;
         defaultgun = _root.p4gun;
         teamnumber = _root.p4team;
   }
}
if(_root.teamgame)
{
   nametag.gotoAndStop(teamnumber + 1);
   nametag.nametext.text = displayname;
}
else
{
   nametag.gotoAndStop(1);
   nametag.nametext.text = displayname;
}
triplejump = false;
currentlevel = 0;
UPGRADE();
currentgun = 2;
respawn();
targettime = 0;
groundleft = _root.ground._x + _root.ground.platform._x;
groundright = _root.ground._x + _root.ground.platform._x + _root.ground.platform._width;
groundmiddle = (groundleft + groundright) / 2;
lockleft = 0;
lockright = 0;
lockup = 0;
cratearray = new Array();
targetplayer = false;
idletime2 = 0;
prevx = 0;
invisibletime = 0;
shieldtime = 0;
jetfuel = 0;
speedtime = 0;
minitime = 0;
minimulti = 1;
playerspeed = _root.speed;
dummy = false;
if(_root.gototest)
{
   dummy = true;
}
killself = false;
weight = 1;
damagemulti = 2 - _root.zombiewave * 0.05;
playerspeed *= 0.3 + _root.zombiewave * 0.066;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(_root.gamewin)
      {
         vx = 0;
         vy = 0;
      }
      targettime += 1;
      if(targettime >= 40)
      {
         low = 5000;
         i = 0;
         while(i < _root.activeplayers.length)
         {
            if(activeplayers[i].iszombie)
            {
               break;
            }
            if(_root.activeplayers[i].teamnumber != teamnumber)
            {
               if(_root.activeplayers[i].PLAYERNUMBER != this.PLAYERNUMBER)
               {
                  distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
                  if(distance < low)
                  {
                     low = distance;
                     target = _root.activeplayers[i];
                     targetplayer = true;
                  }
               }
            }
            if(i == 3)
            {
               break;
            }
            i++;
         }
         targettime = 0;
      }
      _X = _X + vx;
      _Y = _Y + vy;
      if(!freepass)
      {
         if(_root.ground.platform.hitTest(_X,_Y,true) && vy > 0)
         {
            if(Math.abs(vx) < 3)
            {
               gothitby = "none";
               hitbynade = false;
               greedykill = false;
            }
            jumpnum = 2;
            if(perknumber == 3)
            {
               triplejump = true;
            }
            _Y = _Y - vy * 1.01;
            i = 1;
            while(i <= 5)
            {
               if(_root.ground.platform.hitTest(_X,_Y + i * (vy / 5),true))
               {
                  _Y = _Y + vy / 5 * (i - 0.5);
                  break;
               }
               i++;
            }
            vy = 0;
         }
         else if(jumpnum == 2)
         {
            jumpnum = 1;
         }
      }
      if(!_root.ground.platform.hitTest(_X,_Y - 8,true) && !_root.ground.platform.hitTest(_X,_Y,true))
      {
         freepass = false;
      }
      if(_root.ground.platform.hitTest(_X,_Y - 8,true) && !freepass)
      {
         freepass = true;
      }
      if(_Y > 1000 || _X < -600 || _X > 1500)
      {
         gotkilled();
      }
      land1x = _X + land1._x * 0.8;
      land1y = _Y + land1._y * 0.8;
      land2x = _X + land2._x * 0.8;
      land2y = _Y + land2._y * 0.8;
      noland = _Y + noland._y * 0.8;
      if(prevx == Math.round(_X) && !dummy)
      {
         idletime2 += 1;
         if(idletime2 >= 4)
         {
            idletime2 = 0;
            if(jumpnum == 2)
            {
               KEYUP = true;
            }
            if(target._x < _X)
            {
               lockleft = 10;
            }
            if(target._x >= _X)
            {
               lockright = 10;
            }
         }
      }
      else
      {
         idletime2 = 0;
      }
      prevx = Math.round(_X);
      if(target._y <= _Y + 10 && target._y >= _Y - 80 && jumpnum == 2)
      {
         if(targetplayer)
         {
            if(target._x >= groundmiddle)
            {
               dir = 1;
            }
            if(target._x < groundmiddle)
            {
               dir = -1;
            }
            optimalx = target._x;
         }
         else
         {
            optimalx = target._x;
         }
         if(_X > groundleft + 200)
         {
            if(optimalx < _X - 40)
            {
               KEYLEFT = true;
            }
         }
         if(_X < groundright - 200)
         {
            if(optimalx > _X + 40)
            {
               KEYRIGHT = true;
            }
         }
         if(_X > optimalx - 40 && _X < optimalx + 40)
         {
            if(target._x > _X && facing == -1)
            {
               KEYRIGHT = true;
            }
            if(target._x <= _X && facing == 1)
            {
               KEYLEFT = true;
            }
         }
      }
      else if(target._y > _Y + 10)
      {
         if(_X > groundleft + 50 && target.jumpnum == 2 || _X > groundleft - 10 && !targetplayer)
         {
            if(target._x < _X - 30)
            {
               KEYLEFT = true;
            }
         }
         if(_X < groundright - 50 && target.jumpnum == 2 || _X < groundright + 10 && !targetplayer)
         {
            if(target._x > _X + 30)
            {
               KEYRIGHT = true;
            }
         }
         if(_X > target._x - 30 && _X < target._x + 30 && Math.round(target.vy) == 1 && target.jumpnum == 2 || _X > target._x - 30 && _X < target._x + 30 && !targetplayer)
         {
            KEYDOWN = true;
         }
         if(_root.ground.platform.hitTest(_X,_Y + 95,true) && target.jumpnum == 2)
         {
            KEYDOWN = true;
         }
      }
      else if(target._y < _Y - 80)
      {
         if(vy < 0 && jumpnum == 1 && _root.ground.platform.hitTest(_X,_Y - 30,true) && Math.abs(vx) <= 5)
         {
            KEYUP = true;
         }
         if(jumpnum == 2 && _root.ground.platform.hitTest(_X,_Y - 75,true) && Math.abs(vx) <= 5 || jumpnum == 2 && _root.ground.platform.hitTest(_X,_Y - 120,true) && Math.abs(vx) <= 5)
         {
            KEYUP = true;
         }
         else if(jumpnum == 2 && _root.ground.platform.hitTest(_X - 100,_Y - 80,true) && lockleft == 0 && vx >= -5 || jumpnum == 2 && _root.ground.platform.hitTest(_X - 100,_Y - 120,true) && lockleft == 0 && vx >= -5)
         {
            KEYUP = true;
            lockleft = 20;
         }
         else if(jumpnum == 2 && _root.ground.platform.hitTest(_X + 100,_Y - 80,true) && lockright == 0 && vx <= 5 || jumpnum == 2 && _root.ground.platform.hitTest(_X + 100,_Y - 120,true) && lockright == 0 && vx <= 5)
         {
            KEYUP = true;
            lockright = 20;
         }
         else if(jumpnum == 2)
         {
            if(target._x < _X && lockright == 0)
            {
               lockleft = 10;
            }
            else if(target._x > _X && lockleft == 0)
            {
               lockright = 10;
            }
         }
      }
      if(_X < groundleft + 100 || _X > groundright - 100)
      {
         if(groundmiddle < _X - 40)
         {
            KEYLEFT = true;
         }
         if(groundmiddle > _X + 40)
         {
            KEYRIGHT = true;
         }
      }
      if(Math.abs(vx) > 30)
      {
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
      }
      if(vx > 15 || vx < -15)
      {
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
      }
      if(!_root.ground.platform.hitTest(land2x,land2y,true) && vx > 10)
      {
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
      }
      if(!_root.ground.platform.hitTest(land1x,land1y,true) && vx < -10)
      {
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
      }
      if(_X > groundright - 100 && targetplayer || _X > groundright && !targetplayer)
      {
         if(jetfuel > 0 && _X > groundright && vy > -1)
         {
            lockup = 10;
         }
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
         if(vy > 0 && vx < -1 && jumpnum == 1 && _X < groundright + 100 || vy > 0 && vx < -1 && jumpnum == 11 && _X < groundright + 100)
         {
            KEYUP = true;
         }
         if(_X > groundright && !_root.ground.platform.hitTest(_X,noland,true) && jumpnum == 1 && _Y > _root.ground._y + _root.ground.lowest._y - 100 || jumpnum == 11 && _X > groundright && !_root.ground.platform.hitTest(_X,noland,true) && _Y > _root.ground._y + _root.ground.lowest._y - 100)
         {
            KEYUP = true;
         }
      }
      if(_X < groundleft + 100 && targetplayer || _X < groundleft && !targetplayer)
      {
         if(jetfuel > 0 && _X < groundleft && vy > -1)
         {
            lockup = 10;
         }
         if(jumpnum == 2)
         {
            KEYUP = true;
         }
         if(vy > 0 && vx > 1 && jumpnum == 1 && _X > groundleft - 100 || vy > 0 && vx > 1 && jumpnum == 11 && _X > groundleft - 100)
         {
            KEYUP = true;
         }
         if(_X < groundleft && !_root.ground.platform.hitTest(_X,noland,true) && jumpnum == 1 && _Y > _root.ground._y + _root.ground.lowest._y - 100 || jumpnum == 11 && _X < groundleft && !_root.ground.platform.hitTest(_X,noland,true) && _Y > _root.ground._y + _root.ground.lowest._y - 100)
         {
            KEYUP = true;
         }
      }
      if(!_root.ground.platform.hitTest(land2x,land2y,true) && jumpnum == 2)
      {
         KEYRIGHT = false;
      }
      if(!_root.ground.platform.hitTest(land1x,land1y,true) && jumpnum == 2)
      {
         KEYLEFT = false;
      }
      if(!_root.ground.platform.hitTest(land2x - 15,land2y,true) && jumpnum == 2)
      {
         KEYRIGHT = false;
         KEYLEFT = true;
      }
      if(!_root.ground.platform.hitTest(land1x + 15,land1y,true) && jumpnum == 2)
      {
         KEYLEFT = false;
         KEYRIGHT = true;
      }
      if(lockright >= lockleft)
      {
         lockleft = 0;
      }
      if(lockleft > lockright)
      {
         lockright = 0;
      }
      if(lockright > 0)
      {
         KEYRIGHT = true;
         lockright -= 1;
      }
      if(lockleft > 0)
      {
         KEYLEFT = true;
         lockleft -= 1;
      }
      if(lockup > 0)
      {
         KEYUP = true;
         lockup -= 1;
      }
      if(targetplayer && target._y < _Y + 20 && target._y > _Y - 80)
      {
         if(target._x > _X && facing == 1 || target._x < _X && facing == -1)
         {
            if(hand1.gun.shotgun > 0)
            {
               if(Math.abs(target._x - _X) > 15 && Math.abs(target._x - _X) < 150)
               {
               }
            }
         }
      }
      if(!disabled && !_root.gamewin)
      {
         if(KEYDOWN && jumpnum == 2 && !freepass && !downkey)
         {
            if(_Y < _root.ground._y + _root.ground.lowest._y)
            {
               freepass = true;
               vy += 1;
               _Y = _Y + 5;
               jumpnum = 1;
               downkey = true;
            }
         }
         else if(!KEYDOWN && downkey)
         {
            downkey = false;
         }
         if(KEYUP && !jumpkey && jumpnum > 0)
         {
            jumpkey = true;
            jumpnum -= 1;
            if(jumpnum == 1)
            {
               vy = (- _root.power) * 1;
            }
            else if(jumpnum == 0)
            {
               vy = (- _root.power) * 0.83;
               _root.CP("fx_double",_X,_Y);
               if(triplejump && jumpnum == 0)
               {
                  jumpnum = 11;
                  triplejump = false;
               }
            }
            if(jumpnum == 10)
            {
               jumpnum = 0;
               if(vy < -4)
               {
                  vy -= _root.power * 0.2;
               }
               else
               {
                  vy = (- _root.power) * 0.65;
               }
               _root.CP("fx_double",_X,_Y);
            }
            _Y = _Y - 1;
         }
         if(!KEYUP)
         {
            jumpkey = false;
         }
         if(KEYLEFT)
         {
            startwalk();
            if(jumpnum == 2)
            {
               vx -= playerspeed * weight;
            }
            else if(_root.ground.lowfriction)
            {
               vx -= playerspeed / 1.4;
            }
            else
            {
               vx -= playerspeed / 1.1 * weight;
            }
            if(facing == 1)
            {
               leg1._rotation *= -1;
               leg2._rotation *= -1;
            }
            facing = -1;
         }
         if(KEYRIGHT)
         {
            startwalk();
            if(jumpnum == 2)
            {
               vx += playerspeed * weight;
            }
            else if(_root.ground.lowfriction)
            {
               vx += playerspeed / 1.4;
            }
            else
            {
               vx += playerspeed / 1.1 * weight;
            }
            if(facing == -1)
            {
               leg1._rotation *= -1;
               leg2._rotation *= -1;
            }
            facing = 1;
         }
      }
      if(_root.gamewin && perknumber == 1)
      {
         if(Key.isDown(32))
         {
            body._rotation += (-15 * facing - body._rotation) / 2;
            hand1._x += (-10 * facing - hand1._x) / 2;
            hand2._x += (-10 * facing - hand2._x) / 2;
         }
         else
         {
            body._rotation += (- body._rotation) / 3;
            hand1._x += (- hand1._x) / 3;
            hand2._x += (- hand2._x) / 3;
         }
      }
      if(!_root.gamewin && _root.gamemode != 2)
      {
         if(offhandnumber == 1 && offhandammo > 0)
         {
            if(KEYNADE && !nadekey)
            {
               nadekey = true;
               hand2.gotoAndPlay(2);
               nadepower = 1;
            }
            else if(!KEYNADE && nadekey)
            {
               nadekey = false;
               _root.CP("wep_grenade",_X + 30 * facing,_Y - 35,facing,nadepower + PLAYERNUMBER * 1000);
               offhandammo -= 1;
               hand2.gotoAndPlay(6);
            }
            if(KEYNADE)
            {
               if(nadepower < 14)
               {
                  nadepower += 0.5;
               }
            }
         }
         else if(offhandnumber == 2 && offhandammo > 0)
         {
            if(KEYNADE && !nadekey)
            {
               nadekey = true;
               if(hand2._currentframe == 1)
               {
                  hand2.gotoAndPlay(20);
               }
            }
            else if(KEYNADE && nadekey)
            {
               nadekey = false;
            }
         }
         else if(offhandnumber == 3)
         {
            if(KEYNADE && !nadekey)
            {
               nadekey = true;
               if(hand2._currentframe == 1)
               {
                  hand2.gotoAndPlay(40);
               }
            }
            else if(KEYNADE && nadekey)
            {
               nadekey = false;
            }
         }
      }
      vx *= _root.friction;
      if(vy < 24)
      {
         vy += _root.gravity;
      }
      if(Math.abs(vx) <= 0.1)
      {
         vx = 0;
      }
      if(perknumber == 2 && weight != 1)
      {
         weight = 1;
      }
      if(perknumber == 5 && recoil != 0)
      {
         recoil = 0;
      }
      if(jetfuel > 0)
      {
         jetfuel -= 0.12;
      }
      KEYUP = false;
      KEYDOWN = false;
      KEYLEFT = false;
      KEYRIGHT = false;
      KEYSHOOT = false;
      KEYNADE = false;
      if(killself)
      {
         SELFDESTRUCT();
      }
      if(_root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
      ducky._xscale = 100 * facing;
      i = 0;
      while(i < _root.activeplayers.length)
      {
         if(_root.activeplayers[i].iszombie)
         {
            break;
         }
         if(_root.activeplayers[i].frame.hitTest(_X,_Y - 15,true))
         {
            if(_root.activeplayers[i].shieldtime <= 0)
            {
               radians = Math.atan2(_root.activeplayers[i]._y - 30 - _Y,_root.activeplayers[i]._x - _X);
               degrees = radians * 180 / 3.141592653589793;
               pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 10;
               pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 20;
               _root.activeplayers[i].vx += pushx;
               _root.activeplayers[i].lives -= 1;
               if(_root.activeplayers[i].lives <= 0)
               {
                  _root.activeplayers[i].SELFDESTRUCT();
               }
               if(!isNaN(pushy))
               {
                  _root.activeplayers[i].vy += pushy;
               }
               j = 0;
               while(j < 4)
               {
                  _root.CP("fx_blood",_X,_Y,0,0);
                  if(_root.activeplayers[i].iszombie && j == 1)
                  {
                     break;
                  }
                  j++;
               }
               _root.CP("fx_bloodstain",_X,_Y,0,0);
            }
            _root.CP("fx_ex66",_X,_Y - 20,0);
            _root.CP("fx_ex22",_X,_Y - 20,0);
            _root.CP("fx_ex0",_X,_Y - 20,0);
            _root.CP("fx_ex44",_X,_Y - 20,0);
            i = 0;
            while(i < 2)
            {
               _root.CP("fx_ex5",_X,_Y - 20,0);
               i++;
            }
            i = 0;
            while(i < 2)
            {
               _root.CP("fx_ex7",_X,_Y - 20,0);
               i++;
            }
            _root.CP("fx_combo",_X,_Y - 50,0,-3);
            _root.explodesound();
            gotkilled();
         }
         if(i == 3)
         {
            break;
         }
         i++;
      }
   }
};
